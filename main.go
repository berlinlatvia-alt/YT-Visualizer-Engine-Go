package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ─────────────────────────────────────────────────────────────────────────────
// CONFIG & TYPES
// ─────────────────────────────────────────────────────────────────────────────

type ChannelConfig struct {
	Wordmark     string
	Tagline      string
	Subtitle     string
	WordmarkFont string
	BodyFont     string
	Color        string
	Curves       string
}

type Palette struct {
	LOW  string `json:"LOW"`
	MID  string `json:"MID"`
	HIGH string `json:"HIGH"`
	KICK string `json:"KICK"`
}

type JobEvent struct {
	Kind  string `json:"kind"`
	Stage string `json:"stage,omitempty"`
	Pct   int    `json:"pct,omitempty"`
	Url   string `json:"url,omitempty"`
	Ok    bool   `json:"ok,omitempty"`
	Msg   string `json:"msg,omitempty"`
}

type Job struct {
	ID       string
	Kind     string
	Channel  string
	Progress int
	Status   string
	Events   []JobEvent
	Clients  map[chan string]bool
	Mutex    sync.Mutex
	Start    time.Time
}

var (
	PORT      = "42002"
	TOKEN     = "local-dev-token"
	WORK_BASE = "work_local"
	JOBS      = make(map[string]*Job)
	JOBS_MU   sync.RWMutex
)

var CHANNEL_CONFIGS = map[string]ChannelConfig{
	"eva": {
		Wordmark:     "E V A   V I B E",
		Tagline:      "MEDITATION MUSIC  •  DEEP RELAXATION",
		Subtitle:     "FIND YOUR PEACE",
		WordmarkFont: "fonts/Cinzel-Regular.ttf",
		BodyFont:     "fonts/Inter-Medium.ttf",
		Color:        "0xE8D9A8",
		Curves:       "r='0/0 0.25/0.18 0.55/0.62 1/1':g='0/0 0.5/0.48 1/0.97':b='0/0.05 0.4/0.36 0.7/0.62 1/0.92'",
	},
	"vne": {
		Wordmark:     "В Н Е   С И С Т Е М Ы",
		Tagline:      "МЕДИТАТИВНАЯ МУЗЫКА  •  ГЛУБОКИЙ ТРАНС",
		Subtitle:     "НАЙДИ СЕБЯ",
		WordmarkFont: "fonts/PTSerif-Regular.ttf",
		BodyFont:     "fonts/Inter-Medium.ttf",
		Color:        "0xC8B8E8",
		Curves:       "r='0/0 0.5/0.42 1/0.88':g='0/0 0.5/0.48 1/0.94':b='0/0.08 0.3/0.32 0.7/0.74 1/1'",
	},
}

var EQ_BRAND = map[string]Palette{
	"eva": {LOW: "0xC25E2D", MID: "0xF5C89A", HIGH: "0x8C4A28", KICK: "0xF5C89A"},
	"vne": {LOW: "0x5A2A6E", MID: "0x9A85C5", HIGH: "0x3A2858", KICK: "0x9A85C5"},
}

// ─────────────────────────────────────────────────────────────────────────────
// UTILS
// ─────────────────────────────────────────────────────────────────────────────

func ffEscape(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	return strings.ReplaceAll(p, ":", "\\:")
}

func getJob(id string) *Job {
	JOBS_MU.RLock()
	defer JOBS_MU.RUnlock()
	return JOBS[id]
}

func createJob(kind, channel string) *Job {
	id := uuid.New().String()
	job := &Job{
		ID:      id,
		Kind:    kind,
		Channel: channel,
		Status:  "running",
		Clients: make(map[chan string]bool),
		Start:   time.Now(),
	}
	JOBS_MU.Lock()
	JOBS[id] = job
	JOBS_MU.Unlock()
	return job
}

func (j *Job) PushEvent(ev JobEvent) {
	j.Mutex.Lock()
	defer j.Mutex.Unlock()
	j.Events = append(j.Events, ev)
	data, _ := json.Marshal(ev)
	msg := fmt.Sprintf("data: %s\n\n", string(data))
	for client := range j.Clients {
		client <- msg
	}
}

func (j *Job) Finish(ok bool, msg string) {
	j.Mutex.Lock()
	if ok {
		j.Status = "done"
	} else {
		j.Status = "error"
	}
	j.Mutex.Unlock()
	j.PushEvent(JobEvent{Kind: "end", Ok: ok, Msg: msg})
	j.Mutex.Lock()
	for client := range j.Clients {
		close(client)
		delete(j.Clients, client)
	}
	j.Mutex.Unlock()
}

// ─────────────────────────────────────────────────────────────────────────────
// ASSET PREPARATION
// ─────────────────────────────────────────────────────────────────────────────

func extractPalette(imagePath string) (Palette, error) {
	ppmPath := imagePath + ".tmp.ppm"
	cmd := exec.Command("ffmpeg", "-y", "-i", imagePath, "-vf", "scale=3:1:flags=area+accurate_rnd,format=rgb24", ppmPath)
	if err := cmd.Run(); err != nil {
		return EQ_BRAND["eva"], err
	}
	defer os.Remove(ppmPath)

	data, err := os.ReadFile(ppmPath)
	if err != nil {
		return EQ_BRAND["eva"], err
	}

	// Simple PPM parsing (skip 3 headers)
	pos := 0
	nl := 0
	for nl < 3 && pos < len(data) {
		if data[pos] == 10 {
			nl++
		}
		pos++
	}

	type color struct {
		hex string
		lum float64
	}
	colors := make([]color, 3)
	for i := 0; i < 3; i++ {
		r, g, b := float64(data[pos+i*3]), float64(data[pos+i*3+1]), float64(data[pos+i*3+2])
		lum := 0.299*r + 0.587*g + 0.114*b
		hex := fmt.Sprintf("0x%02x%02x%02x", int(r), int(g), int(b))
		colors[i] = color{hex, lum}
	}

	sort.Slice(colors, func(i, j int) bool {
		return colors[i].lum < colors[j].lum
	})

	return Palette{
		LOW:  colors[0].hex,
		MID:  colors[2].hex,
		HIGH: colors[1].hex,
		KICK: EQ_BRAND["eva"].KICK,
	}, nil
}

func generateGlow(imagePath, glowPath string) (string, error) {
	cmd := exec.Command("ffmpeg", "-y", "-i", imagePath, "-vf",
		"scale=1920:1080:force_original_aspect_ratio=decrease,pad=1920:1080:(ow-iw)/2:(oh-ih)/2:color=black,hue=s=1.3,gblur=sigma=20:steps=2,eq=saturation=1.3:gamma=1.1:brightness=0.06",
		"-frames:v", "1", glowPath)
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return glowPath, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// FILTER BUILDER
// ─────────────────────────────────────────────────────────────────────────────

func buildFilter(n int, palette Palette, channel string, glowPath string) string {
	cfg := CHANNEL_CONFIGS[channel]
	eqBrand := EQ_BRAND[channel]
	eqCol := fmt.Sprintf("%s|%s|%s|%s", eqBrand.LOW, eqBrand.MID, eqBrand.HIGH, eqBrand.LOW)
	wf := ffEscape(cfg.WordmarkFont)
	bf := ffEscape(cfg.BodyFont)

	var parts []string

	// 1. Audio Concat
	for i := 0; i < n; i++ {
		parts = append(parts, fmt.Sprintf("[%d:a]aformat=sample_rates=48000:sample_fmts=fltp:channel_layouts=stereo,asetpts=N/SR/TB[a%d]", i, i))
	}
	var aInputs []string
	for i := 0; i < n; i++ {
		aInputs = append(aInputs, fmt.Sprintf("[a%d]", i))
	}
	parts = append(parts, fmt.Sprintf("%sconcat=n=%d:v=0:a=1[a_concat]", strings.Join(aInputs, ""), n))
	parts = append(parts, "[a_concat]asplit=2[a_out][a_eq]")

	// 2. Equalizer
	parts = append(parts, fmt.Sprintf("[a_eq]showfreqs=s=960x280:mode=bar:ascale=log:fscale=log:win_size=4096:averaging=3:colors=%s,fps=30,format=yuva420p[eq_half]", eqCol))
	parts = append(parts, "[eq_half]split[eq_a][eq_b]")
	parts = append(parts, "[eq_b]hflip[eq_b_flip]")
	parts = append(parts, "[eq_b_flip][eq_a]hstack=inputs=2,format=yuva420p,colorchannelmixer=aa=0.85,gblur=sigma=1.2[eq]")

	// 3. Background
	coverIdx := n
	parts = append(parts, fmt.Sprintf("[%d:v]scale=1920:1080:force_original_aspect_ratio=decrease,pad=1920:1080:(ow-iw)/2:(oh-ih)/2:color=black,setsar=1,format=yuv420p[bg_raw]", coverIdx))

	if glowPath != "" {
		glowIdx := n + 1
		parts = append(parts, fmt.Sprintf("[bg_raw][%d:v]blend=all_mode=screen:all_opacity=0.28[bg_glow]", glowIdx))
	} else {
		parts = append(parts, "[bg_raw]copy[bg_glow]")
	}

	parts = append(parts, fmt.Sprintf("[bg_glow]curves=%s,eq=saturation=0.88:contrast=1.06,vignette=PI/5:eval=init,format=yuv420p[bg_graded]", cfg.Curves))
	parts = append(parts, "[bg_graded]noise=alls=28:allf=t+u,format=yuv420p[bg_grained]")

	// Title Block
	parts = append(parts, fmt.Sprintf("[bg_grained]drawbox=x=0:y=0:w=iw:h=ih*0.16:color=black@0.55:t=fill,"+
		"drawtext=fontfile='%s':text='%s':fontcolor=%s:fontsize=30:x=(w-text_w)/2:y=h*0.015:expansion=normal,"+
		"drawtext=fontfile='%s':text='%s':fontcolor=%s:fontsize=72:x=(w-text_w)/2:y=h*0.050:shadowcolor=%s@0.35:shadowx=0:shadowy=0,"+
		"drawtext=fontfile='%s':text='%s':fontcolor=%s:fontsize=28:x=(w-text_w)/2:y=h*0.125:expansion=normal,format=yuv420p[bg_final]",
		wf, cfg.Tagline, cfg.Color, wf, cfg.Wordmark, cfg.Color, cfg.Color, bf, cfg.Subtitle, cfg.Color))

	// Final Composite
	parts = append(parts, "[bg_final][eq]overlay=x=0:y=H-h-40:format=auto[bg_with_eq]")

	kick := eqBrand.KICK
	parts = append(parts, fmt.Sprintf("[bg_with_eq]drawbox=x=0:y=ih-43:w=iw:h=3:color=%s@0.95:t=fill,"+
		"drawbox=x=0:y=ih-53:w=iw:h=10:color=%s@0.45:t=fill,"+
		"drawbox=x=0:y=ih-66:w=iw:h=13:color=%s@0.22:t=fill,"+
		"drawbox=x=0:y=ih-84:w=iw:h=18:color=%s@0.10:t=fill[outv]",
		kick, kick, kick, kick))

	return strings.Join(parts, ";")
}

// ─────────────────────────────────────────────────────────────────────────────
// RENDERER
// ─────────────────────────────────────────────────────────────────────────────

func runFfmpeg(job *Job, audioPaths []string, imagePath, glowPath, outPath string, audioDur float64, previewOnly bool) error {
	outputDur := audioDur
	if previewOnly && audioDur > 10 {
		outputDur = 10
	}
	guardDur := outputDur + 2

	args := []string{}
	for _, a := range audioPaths {
		args = append(args, "-i", a)
	}
	args = append(args, "-t", fmt.Sprintf("%f", guardDur), "-loop", "1", "-framerate", "30", "-i", imagePath)
	if glowPath != "" {
		args = append(args, "-t", fmt.Sprintf("%f", guardDur), "-loop", "1", "-framerate", "30", "-i", glowPath)
	}

	filter := buildFilter(len(audioPaths), EQ_BRAND[job.Channel], job.Channel, glowPath)
	args = append(args, "-filter_complex", filter)
	args = append(args, "-map", "[outv]", "-map", "[a_out]")
	args = append(args, "-c:v", "libx264", "-preset", "ultrafast", "-tune", "stillimage", "-crf", "22", "-pix_fmt", "yuv420p", "-r", "30", "-c:a", "aac", "-b:a", "192k", "-ar", "48000")
	args = append(args, "-t", fmt.Sprintf("%f", outputDur))
	args = append(args, "-progress", "pipe:2", "-nostats", "-y", outPath)

	cmd := exec.Command("ffmpeg", args...)
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stderr)
	re := regexp.MustCompile(`out_time_ms=(\d+)`)
	for scanner.Scan() {
		line := scanner.Text()
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			ms, _ := strconv.ParseFloat(matches[1], 64)
			pct := int((ms / 1000000) / outputDur * 100)
			if pct > 99 {
				pct = 99
			}
			job.PushEvent(JobEvent{Kind: "progress", Pct: pct})
		}
	}

	return cmd.Wait()
}

// ─────────────────────────────────────────────────────────────────────────────
// ENDPOINTS
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	r := gin.Default()

	// CORS config
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Auth Middleware
	auth := func(c *gin.Context) {
		token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if token == "" {
			token = c.Query("token")
		}
		if token != TOKEN {
			c.JSON(401, gin.H{"error": "auth required"})
			c.Abort()
			return
		}
		c.Next()
	}

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true, "version": "go-v1", "uptime": time.Since(time.Now()).Seconds()})
	})

	r.GET("/session", auth, func(c *gin.Context) {
		data, err := os.ReadFile(filepath.Join(WORK_BASE, "last_session.json"))
		if err != nil {
			c.JSON(404, nil)
			return
		}
		var sess interface{}
		json.Unmarshal(data, &sess)
		c.JSON(200, sess)
	})

	r.GET("/progress/:jobId", func(c *gin.Context) {
		jobID := c.Param("jobId")
		job := getJob(jobID)
		if job == nil {
			c.Status(404)
			return
		}

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		// Replay past events
		job.Mutex.Lock()
		for _, ev := range job.Events {
			data, _ := json.Marshal(ev)
			c.Writer.WriteString(fmt.Sprintf("data: %s\n\n", string(data)))
		}
		job.Mutex.Unlock()
		c.Writer.Flush()

		if job.Status != "running" {
			return
		}

		ch := make(chan string)
		job.Mutex.Lock()
		job.Clients[ch] = true
		job.Mutex.Unlock()

		defer func() {
			job.Mutex.Lock()
			delete(job.Clients, ch)
			job.Mutex.Unlock()
		}()

		c.Stream(func(w io.Writer) bool {
			msg, ok := <-ch
			if !ok {
				return false
			}
			w.Write([]byte(msg))
			return true
		})
	})

	r.GET("/previews/:jobId", auth, func(c *gin.Context) {
		jobID := c.Param("jobId")
		path := filepath.Join(WORK_BASE, jobID, "preview.mp4")
		c.File(path) // Gin handles range requests automatically
	})

	// Upload handler (Preview & Full)
	r.POST("/preview", auth, func(c *gin.Context) {
		job := createJob("preview", c.PostForm("channel"))
		if job.Channel == "" {
			job.Channel = "eva"
		}

		c.JSON(200, gin.H{"jobId": job.ID})

		// Helper to save session
		saveSess := func(img string, auds []string) {
			sess := map[string]interface{}{
				"imagePath":  img,
				"imageName":  filepath.Base(img),
				"audioPaths": auds,
				"audioNames": func() []string {
					var names []string
					for _, p := range auds {
						names = append(names, filepath.Base(p))
					}
					return names
				}(),
				"savedAt": time.Now().Format(time.RFC3339),
			}
			b, _ := json.MarshalIndent(sess, "", "  ")
			os.MkdirAll(WORK_BASE, 0755)
			os.WriteFile(filepath.Join(WORK_BASE, "last_session.json"), b, 0644)
		}

		go func() {
			jobDir := filepath.Join(WORK_BASE, job.ID)
			os.MkdirAll(jobDir, 0755)

			form, err := c.MultipartForm()
			if err != nil {
				job.Finish(false, "Invalid form data")
				return
			}
			image := form.File["image"][0]
			imagePath := filepath.Join(jobDir, image.Filename)
			c.SaveUploadedFile(image, imagePath)

			var audioPaths []string
			for i, a := range form.File["audio"] {
				p := filepath.Join(jobDir, fmt.Sprintf("track_%d_%s", i, a.Filename))
				c.SaveUploadedFile(a, p)
				audioPaths = append(audioPaths, p)
			}

			saveSess(imagePath, audioPaths)

			// Extract assets
			job.PushEvent(JobEvent{Kind: "stage", Stage: "Extracting palette & glow..."})
			palette, _ := extractPalette(imagePath)
			glowPath, _ := generateGlow(imagePath, filepath.Join(jobDir, "glow.png"))

			job.PushEvent(JobEvent{Kind: "stage", Stage: "Rendering 10s preview..."})
			err = runFfmpeg(job, audioPaths, imagePath, glowPath, filepath.Join(jobDir, "preview.mp4"), 10, true)
			if err != nil {
				job.Finish(false, err.Error())
				return
			}

			job.PushEvent(JobEvent{Kind: "progress", Pct: 100})
			job.PushEvent(JobEvent{Kind: "preview-ready", Url: "/previews/" + job.ID})
			job.Finish(true, "Preview ready")
		}()
	})

	r.POST("/preview-saved", auth, func(c *gin.Context) {
		data, _ := os.ReadFile(filepath.Join(WORK_BASE, "last_session.json"))
		var sess struct {
			ImagePath  string   `json:"imagePath"`
			AudioPaths []string `json:"audioPaths"`
		}
		json.Unmarshal(data, &sess)

		job := createJob("preview", c.PostForm("channel"))
		c.JSON(200, gin.H{"jobId": job.ID})

		go func() {
			jobDir := filepath.Join(WORK_BASE, job.ID)
			os.MkdirAll(jobDir, 0755)
			palette, _ := extractPalette(sess.ImagePath)
			glowPath, _ := generateGlow(sess.ImagePath, filepath.Join(jobDir, "glow.png"))
			runFfmpeg(job, sess.AudioPaths, sess.ImagePath, glowPath, filepath.Join(jobDir, "preview.mp4"), 10, true)
			job.PushEvent(JobEvent{Kind: "preview-ready", Url: "/previews/" + job.ID})
			job.Finish(true, "Preview ready")
		}()
	})

	r.POST("/render-full", auth, func(c *gin.Context) {
		job := createJob("render", c.PostForm("channel"))
		c.JSON(200, gin.H{"jobId": job.ID})

		go func() {
			jobDir := filepath.Join(WORK_BASE, job.ID)
			os.MkdirAll(jobDir, 0755)
			form, _ := c.MultipartForm()
			image := form.File["image"][0]
			imagePath := filepath.Join(jobDir, image.Filename)
			c.SaveUploadedFile(image, imagePath)
			var audioPaths []string
			for i, a := range form.File["audio"] {
				p := filepath.Join(jobDir, fmt.Sprintf("track_%d_%s", i, a.Filename))
				c.SaveUploadedFile(a, p)
				audioPaths = append(audioPaths, p)
			}

			palette, _ := extractPalette(imagePath)
			glowPath, _ := generateGlow(imagePath, filepath.Join(jobDir, "glow.png"))

			// Get total duration
			var totalDur float64
			for _, a := range audioPaths {
				cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0", a)
				out, _ := cmd.Output()
				d, _ := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
				totalDur += d
			}

			job.PushEvent(JobEvent{Kind: "stage", Stage: fmt.Sprintf("Rendering %.1f min mix...", totalDur/60)})
			runFfmpeg(job, audioPaths, imagePath, glowPath, filepath.Join(jobDir, "output.mp4"), totalDur, false)
			job.Finish(true, "Render complete")
		}()
	})

	r.Static("/", "./dist")

	log.Printf("Go Visualizer Backend running on port %s", PORT)
	r.Run(":" + PORT)
}
