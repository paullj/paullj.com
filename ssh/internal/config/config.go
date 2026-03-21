package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration wraps time.Duration for YAML unmarshaling.
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = dur
	return nil
}

func (d Duration) MarshalYAML() (interface{}, error) {
	return d.String(), nil
}

type Link struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

type ContentConfig struct {
	Name             string `yaml:"name"`
	Subtitle         string `yaml:"subtitle"`
	Description      string `yaml:"description"`
	RecentPostsLimit int    `yaml:"recent_posts_limit"`
	Links            []Link `yaml:"links"`
	AboutPath        string `yaml:"about_path"`
	PostsDir         string `yaml:"posts_dir"`
}

type ImagesConfig struct {
	MaxSize       int      `yaml:"max_size"`
	FetchTimeout  Duration `yaml:"fetch_timeout"`
	CacheMaxBytes int      `yaml:"cache_max_bytes"`
	MaxAsciiWidth int      `yaml:"max_ascii_width"`
}

type FilterConfig struct {
	MaxConcurrentPerIP int      `yaml:"max_concurrent_per_ip"`
	BlockedPrefixes    []string `yaml:"blocked_prefixes"`
}

type SplashConfig struct {
	Text       string   `yaml:"text"`
	CharDelay  Duration `yaml:"char_delay"`
	CharJitter Duration `yaml:"char_jitter"`
	HoldTime   Duration `yaml:"hold_time"`
	SkipOnKey  bool     `yaml:"skip_on_key"`
}

type SSHConfig struct {
	Host        string       `yaml:"host"`
	Port        string       `yaml:"port"`
	HostKeyPath string       `yaml:"host_key_path"`
	IdleTimeout Duration     `yaml:"idle_timeout"`
	MaxTimeout  Duration     `yaml:"max_timeout"`
	RateLimit   Duration     `yaml:"rate_limit"`
	RateBurst   int          `yaml:"rate_burst"`
	MaxWidth    int          `yaml:"max_width"`
	Images      ImagesConfig `yaml:"images"`
	Splash      SplashConfig `yaml:"splash"`
	Filter      FilterConfig `yaml:"filter"`
}

type Config struct {
	Content ContentConfig `yaml:"content"`
	SSH     SSHConfig     `yaml:"ssh"`
}

func Default() *Config {
	return &Config{
		Content: ContentConfig{
			Name:             "Blog Posts",
			Subtitle:         "A blog served over SSH",
			Description:      "A blog served over SSH",
			RecentPostsLimit: 5,
			AboutPath:        "content/about.md",
			PostsDir:         "content/posts",
		},
		SSH: SSHConfig{
			Host:        "0.0.0.0",
			Port:        "2222",
			HostKeyPath: "id_ed25519",
			IdleTimeout: Duration{5 * time.Minute},
			MaxTimeout:  Duration{30 * time.Minute},
			RateLimit:   Duration{time.Second},
			RateBurst:   3,
			MaxWidth:    120,
			Images: ImagesConfig{
				MaxSize:       5 << 20,
				FetchTimeout:  Duration{10 * time.Second},
				CacheMaxBytes: 50 << 20,
				MaxAsciiWidth: 80,
			},
			Splash: SplashConfig{
				Text:       "paullj",
				CharDelay:  Duration{150 * time.Millisecond},
				CharJitter: Duration{50 * time.Millisecond},
				HoldTime:   Duration{800 * time.Millisecond},
				SkipOnKey:  true,
			},
			Filter: FilterConfig{
				MaxConcurrentPerIP: 3,
				BlockedPrefixes:    []string{"SSH-2.0-Go", "libssh"},
			},
		},
	}
}

func Load(path string) (*Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			applyEnvOverrides(cfg)
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	applyEnvOverrides(cfg)
	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	envStr := func(key string, target *string) {
		if v := os.Getenv(key); v != "" {
			*target = v
		}
	}
	envInt := func(key string, target *int) {
		if v := os.Getenv(key); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				*target = n
			}
		}
	}
	envDuration := func(key string, target *Duration) {
		if v := os.Getenv(key); v != "" {
			if d, err := time.ParseDuration(v); err == nil {
				*target = Duration{d}
			}
		}
	}
	envStringSlice := func(key string, target *[]string) {
		if v := os.Getenv(key); v != "" {
			*target = strings.Split(v, ",")
		}
	}

	envStr("PAULLJ_CONTENT_NAME", &cfg.Content.Name)
	envStr("PAULLJ_CONTENT_DESCRIPTION", &cfg.Content.Description)
	envStr("PAULLJ_CONTENT_POSTS_DIR", &cfg.Content.PostsDir)

	envStr("PAULLJ_SSH_HOST", &cfg.SSH.Host)
	envStr("PAULLJ_SSH_PORT", &cfg.SSH.Port)
	envStr("PAULLJ_SSH_HOST_KEY_PATH", &cfg.SSH.HostKeyPath)
	envDuration("PAULLJ_SSH_IDLE_TIMEOUT", &cfg.SSH.IdleTimeout)
	envDuration("PAULLJ_SSH_MAX_TIMEOUT", &cfg.SSH.MaxTimeout)
	envDuration("PAULLJ_SSH_RATE_LIMIT", &cfg.SSH.RateLimit)
	envInt("PAULLJ_SSH_RATE_BURST", &cfg.SSH.RateBurst)
	envInt("PAULLJ_SSH_MAX_WIDTH", &cfg.SSH.MaxWidth)

	envInt("PAULLJ_SSH_IMAGES_MAX_SIZE", &cfg.SSH.Images.MaxSize)
	envDuration("PAULLJ_SSH_IMAGES_FETCH_TIMEOUT", &cfg.SSH.Images.FetchTimeout)
	envInt("PAULLJ_SSH_IMAGES_CACHE_MAX_BYTES", &cfg.SSH.Images.CacheMaxBytes)
	envInt("PAULLJ_SSH_IMAGES_MAX_ASCII_WIDTH", &cfg.SSH.Images.MaxAsciiWidth)

	envInt("PAULLJ_SSH_FILTER_MAX_CONCURRENT_PER_IP", &cfg.SSH.Filter.MaxConcurrentPerIP)
	envStringSlice("PAULLJ_SSH_FILTER_BLOCKED_PREFIXES", &cfg.SSH.Filter.BlockedPrefixes)
}
