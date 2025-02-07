package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"html"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Session struct{}

// poorly named function that fetches active sessions, not videos.
func (c Config) fetch_videos() ([]Video, error) {
	statusURL, err := url.Parse(
		fmt.Sprintf("http://%s:32400/status/sessions?X-Plex-Token=%s", c.PlexIP, c.PlexToken),
	)
	if err != nil {
		return []Video{}, fmt.Errorf("failed to parse status URL: %w", err)
	}
	resp, err := http.Get(statusURL.String())
	if err != nil {
		return []Video{}, fmt.Errorf("failed to issue GET request against status url: %w", err)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return []Video{}, fmt.Errorf("invalid plex token: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return []Video{}, fmt.Errorf("unexpected status code %d return from status url: %w", resp.StatusCode, err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []Video{}, fmt.Errorf("failed to read response body: %w", err)
	}
	ml := MediaContainer{}
	if err = xml.Unmarshal(body, &ml); err != nil {
		return []Video{}, fmt.Errorf("failed to unmarshall response body: %w", err)
	}
	return ml.Videos, nil
}

const filmNotification = `
{{.User.Title}} started watching '{{.Title}}'
`

const tvNotification = `
{{.User.Title}} started watching {{.GrandparentTitle}}: '{{.Title}}'
`

func renderNotification(video Video) (string, error) {
	var tmpl *template.Template
	var err error
	if video.GrandparentTitle != "" {
		tmpl, err = template.New("").Parse(tvNotification)
	} else {
		tmpl, err = template.New("").Parse(filmNotification)
	}
	if err != nil {
		return "", fmt.Errorf("failed creating template: %w", err)
	}
	var renderedMarkdown bytes.Buffer

	err = tmpl.Execute(&renderedMarkdown, video)
	if err != nil {
		return "", fmt.Errorf("failed rendering template: %w", err)
	}
	return html.UnescapeString(renderedMarkdown.String()), nil
}

func (c Config) sendNotification(payload string) error {
	req, err := http.NewRequest(
		http.MethodPost,
		c.NtfyTopicURL,
		strings.NewReader(payload),
	)
	if err != nil {
		return fmt.Errorf("failed to craft request: %w", err)
	}
	req.Header.Set("Title", "New Plex session")
	req.Header.Set("Markdown", "yes")
	req.Header.Set("Tags", "clapper")
	_, err = http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	return nil
}

func (c Config) run() error {
	sessions := make(map[string]bool, 0)
	for {
		videos, err := c.fetch_videos()
		if err != nil {
			return err
		}
		if len(videos) == 0 {
			continue
		}
		for _, video := range videos {
			if video.User.Title == c.IgnoredUser {
				continue
			}
			if sessions[video.SessionKey] { // Already notified
				continue
			}
			sessions[video.SessionKey] = true
			payload, err := renderNotification(videos[0])
			if err != nil {
				return err
			}
			if err = c.sendNotification(payload); err != nil {
				return err
			}
		}
		time.Sleep(c.CheckInterval)
	}
}

type Config struct {
	PlexIP        string        `split_words:"true" required:"true" default:"127.0.0.1"`
	PlexToken     string        `split_words:"true" required:"true"`
	NtfyTopicURL  string        `split_words:"true" required:"true"`
	IgnoredUser   string        `split_words:"true" required:"false"`
	CheckInterval time.Duration `split_words:"true" required:"true" default:"30s"`
}

func exit(err error) {
	fmt.Println(err.Error())
	os.Exit(1)
}

func main() {
	var c Config
	if err := envconfig.Process("psn", &c); err != nil {
		exit(fmt.Errorf("failed to parse env vars: %w", err))
	}
	exit(c.run())
}

type MediaContainer struct {
	XMLName xml.Name `xml:"MediaContainer"`
	Text    string   `xml:",chardata"`
	Size    string   `xml:"size,attr"`
	Videos  []Video  `xml:"Video"`
}

type Video struct {
	XMLName              xml.Name `xml:"Video"`
	Text                 string   `xml:",chardata"`
	AddedAt              string   `xml:"addedAt,attr"`
	Art                  string   `xml:"art,attr"`
	ChapterSource        string   `xml:"chapterSource,attr"`
	ContentRating        string   `xml:"contentRating,attr"`
	Duration             string   `xml:"duration,attr"`
	GrandparentArt       string   `xml:"grandparentArt,attr"`
	GrandparentGuid      string   `xml:"grandparentGuid,attr"`
	GrandparentKey       string   `xml:"grandparentKey,attr"`
	GrandparentRatingKey string   `xml:"grandparentRatingKey,attr"`
	GrandparentSlug      string   `xml:"grandparentSlug,attr"`
	GrandparentThumb     string   `xml:"grandparentThumb,attr"`
	GrandparentTitle     string   `xml:"grandparentTitle,attr"`
	Guid                 string   `xml:"guid,attr"`
	Index                string   `xml:"index,attr"`
	Key                  string   `xml:"key,attr"`
	LastViewedAt         string   `xml:"lastViewedAt,attr"`
	LibrarySectionID     string   `xml:"librarySectionID,attr"`
	LibrarySectionKey    string   `xml:"librarySectionKey,attr"`
	LibrarySectionTitle  string   `xml:"librarySectionTitle,attr"`
	ParentGuid           string   `xml:"parentGuid,attr"`
	ParentIndex          string   `xml:"parentIndex,attr"`
	ParentKey            string   `xml:"parentKey,attr"`
	ParentRatingKey      string   `xml:"parentRatingKey,attr"`
	ParentThumb          string   `xml:"parentThumb,attr"`
	ParentTitle          string   `xml:"parentTitle,attr"`
	RatingKey            string   `xml:"ratingKey,attr"`
	SessionKey           string   `xml:"sessionKey,attr"`
	Thumb                string   `xml:"thumb,attr"`
	Title                string   `xml:"title,attr"`
	Type                 string   `xml:"type,attr"`
	UpdatedAt            string   `xml:"updatedAt,attr"`
	ViewOffset           string   `xml:"viewOffset,attr"`
	Year                 string   `xml:"year,attr"`
	Media                struct {
		Text            string `xml:",chardata"`
		AspectRatio     string `xml:"aspectRatio,attr"`
		AudioChannels   string `xml:"audioChannels,attr"`
		AudioCodec      string `xml:"audioCodec,attr"`
		Bitrate         string `xml:"bitrate,attr"`
		Container       string `xml:"container,attr"`
		Duration        string `xml:"duration,attr"`
		Height          string `xml:"height,attr"`
		ID              string `xml:"id,attr"`
		VideoCodec      string `xml:"videoCodec,attr"`
		VideoFrameRate  string `xml:"videoFrameRate,attr"`
		VideoProfile    string `xml:"videoProfile,attr"`
		VideoResolution string `xml:"videoResolution,attr"`
		Width           string `xml:"width,attr"`
		Selected        string `xml:"selected,attr"`
		Part            struct {
			Text         string `xml:",chardata"`
			Container    string `xml:"container,attr"`
			Duration     string `xml:"duration,attr"`
			File         string `xml:"file,attr"`
			ID           string `xml:"id,attr"`
			Key          string `xml:"key,attr"`
			Size         string `xml:"size,attr"`
			VideoProfile string `xml:"videoProfile,attr"`
			Decision     string `xml:"decision,attr"`
			Selected     string `xml:"selected,attr"`
			Stream       []struct {
				Text                 string `xml:",chardata"`
				BitDepth             string `xml:"bitDepth,attr"`
				Bitrate              string `xml:"bitrate,attr"`
				ChromaLocation       string `xml:"chromaLocation,attr"`
				ChromaSubsampling    string `xml:"chromaSubsampling,attr"`
				Codec                string `xml:"codec,attr"`
				CodedHeight          string `xml:"codedHeight,attr"`
				CodedWidth           string `xml:"codedWidth,attr"`
				Default              string `xml:"default,attr"`
				DisplayTitle         string `xml:"displayTitle,attr"`
				ExtendedDisplayTitle string `xml:"extendedDisplayTitle,attr"`
				FrameRate            string `xml:"frameRate,attr"`
				HasScalingMatrix     string `xml:"hasScalingMatrix,attr"`
				Height               string `xml:"height,attr"`
				ID                   string `xml:"id,attr"`
				Index                string `xml:"index,attr"`
				Language             string `xml:"language,attr"`
				LanguageCode         string `xml:"languageCode,attr"`
				LanguageTag          string `xml:"languageTag,attr"`
				Level                string `xml:"level,attr"`
				Original             string `xml:"original,attr"`
				Profile              string `xml:"profile,attr"`
				RefFrames            string `xml:"refFrames,attr"`
				ScanType             string `xml:"scanType,attr"`
				StreamType           string `xml:"streamType,attr"`
				Width                string `xml:"width,attr"`
				Location             string `xml:"location,attr"`
				AudioChannelLayout   string `xml:"audioChannelLayout,attr"`
				Channels             string `xml:"channels,attr"`
				SamplingRate         string `xml:"samplingRate,attr"`
				Selected             string `xml:"selected,attr"`
			} `xml:"Stream"`
		} `xml:"Part"`
	} `xml:"Media"`
	User struct {
		Text  string `xml:",chardata"`
		ID    string `xml:"id,attr"`
		Thumb string `xml:"thumb,attr"`
		Title string `xml:"title,attr"`
	} `xml:"User"`
	Player struct {
		Text                string `xml:",chardata"`
		Address             string `xml:"address,attr"`
		Device              string `xml:"device,attr"`
		MachineIdentifier   string `xml:"machineIdentifier,attr"`
		Model               string `xml:"model,attr"`
		Platform            string `xml:"platform,attr"`
		PlatformVersion     string `xml:"platformVersion,attr"`
		Product             string `xml:"product,attr"`
		Profile             string `xml:"profile,attr"`
		RemotePublicAddress string `xml:"remotePublicAddress,attr"`
		State               string `xml:"state,attr"`
		Title               string `xml:"title,attr"`
		Vendor              string `xml:"vendor,attr"`
		Version             string `xml:"version,attr"`
		Local               string `xml:"local,attr"`
		Relayed             string `xml:"relayed,attr"`
		Secure              string `xml:"secure,attr"`
		UserID              string `xml:"userID,attr"`
	} `xml:"Player"`
	Session struct {
		Text      string `xml:",chardata"`
		ID        string `xml:"id,attr"`
		Bandwidth string `xml:"bandwidth,attr"`
		Location  string `xml:"location,attr"`
	} `xml:"Session"`
}
