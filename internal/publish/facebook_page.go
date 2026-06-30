package publish

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/squoyster/hydracast/internal/config"
	"github.com/squoyster/hydracast/internal/download"
	"github.com/squoyster/hydracast/internal/secrets"
	"github.com/squoyster/hydracast/internal/source"
)

const (
	fbGraphVersion = "v21.0" // ponytail: bump when FB deprecates; Graph versions are unversioned server-side.
	fbChunkSize    = 8 << 20 // 8MiB; multiple of 1MiB satisfies FB's 256KiB chunk alignment.
	fbGraphHost    = "https://graph.facebook.com"
)

type FacebookPage struct {
	cfg      config.DestinationConfig
	resolver *secrets.Resolver
}

func NewFacebookPage(cfg config.DestinationConfig, resolver *secrets.Resolver) *FacebookPage {
	return &FacebookPage{cfg: cfg, resolver: resolver}
}

func (f *FacebookPage) Name() string { return f.cfg.Name }
func (f *FacebookPage) Type() string { return "facebook_page" }

// Publish uploads via the Facebook Graph API resumable chunked upload
// (start → transfer(loop) → finish). Page videos are public by default.
func (f *FacebookPage) Publish(ctx context.Context, item source.MediaItem, media *download.LocalMedia) (*PublishResult, error) {
	if f.cfg.PageID == "" {
		return &PublishResult{Status: "failed", Error: fmt.Errorf("page_id not configured")}, nil
	}
	if f.cfg.PageTokenRef == "" {
		return &PublishResult{Status: "failed", Error: fmt.Errorf("page_token_ref not configured")}, nil
	}
	pageToken, err := f.resolver.Resolve(f.cfg.PageTokenRef)
	if err != nil {
		return &PublishResult{Status: "failed", Error: fmt.Errorf("resolve page token: %w", err)}, nil
	}

	st, err := os.Stat(media.Path)
	if err != nil {
		return nil, fmt.Errorf("stat media: %w", err)
	}
	fileSize := st.Size()

	file, err := os.Open(media.Path)
	if err != nil {
		return nil, fmt.Errorf("open media: %w", err)
	}
	defer file.Close()

	client := &http.Client{}
	endpoint := fmt.Sprintf("%s/%s/%s/videos", fbGraphHost, fbGraphVersion, f.cfg.PageID)

	// START
	startForm := url.Values{
		"access_token": {pageToken},
		"upload_phase": {"start"},
		"file_size":    {strconv.FormatInt(fileSize, 10)},
	}
	fr, err := fbPostForm(ctx, client, endpoint, startForm)
	if err != nil {
		return &PublishResult{Status: "failed", Error: fmt.Errorf("upload start: %w", err)}, nil
	}
	sessionID := fr.UploadSessionID
	videoID := fr.VideoID
	if sessionID == "" || videoID == "" {
		return &PublishResult{Status: "failed", Error: fmt.Errorf("upload start: missing session_id/video_id")}, nil
	}

	// TRANSFER (chunk loop)
	offset, err := strconv.ParseInt(fr.StartOffset, 10, 64)
	if err != nil {
		return &PublishResult{Status: "failed", Error: fmt.Errorf("parse start_offset %q: %w", fr.StartOffset, err)}, nil
	}
	for offset < fileSize {
		chunkLen := int64(fbChunkSize)
		if rem := fileSize - offset; rem < chunkLen {
			chunkLen = rem
		}

		body := &bytes.Buffer{}
		w := multipart.NewWriter(body)
		_ = w.WriteField("upload_phase", "transfer")
		_ = w.WriteField("access_token", pageToken)
		_ = w.WriteField("upload_session_id", sessionID)
		_ = w.WriteField("start_offset", strconv.FormatInt(offset, 10))
		part, err := w.CreateFormFile("video_file_chunk", filepath.Base(media.Path))
		if err != nil {
			return &PublishResult{Status: "failed", Error: fmt.Errorf("build chunk: %w", err)}, nil
		}
		if _, err := io.Copy(part, io.NewSectionReader(file, offset, chunkLen)); err != nil {
			return &PublishResult{Status: "failed", Error: fmt.Errorf("read chunk: %w", err)}, nil
		}
		if err := w.Close(); err != nil {
			return &PublishResult{Status: "failed", Error: fmt.Errorf("close multipart: %w", err)}, nil
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
		if err != nil {
			return &PublishResult{Status: "failed", Error: fmt.Errorf("chunk request: %w", err)}, nil
		}
		req.Header.Set("Content-Type", w.FormDataContentType())
		tr, err := fbSend(ctx, client, req)
		if err != nil {
			return &PublishResult{Status: "failed", Error: fmt.Errorf("upload transfer @%d: %w", offset, err)}, nil
		}
		next, err := strconv.ParseInt(tr.StartOffset, 10, 64)
		if err != nil {
			return &PublishResult{Status: "failed", Error: fmt.Errorf("parse start_offset %q: %w", tr.StartOffset, err)}, nil
		}
		if next <= offset {
			return &PublishResult{Status: "failed", Error: fmt.Errorf("upload stalled at offset %d (resp %d)", offset, next)}, nil
		}
		offset = next
	}

	// FINISH
	title := item.Title
	if title == "" {
		title = media.Filename
	}
	finishForm := url.Values{
		"access_token":      {pageToken},
		"upload_phase":      {"finish"},
		"upload_session_id": {sessionID},
		"title":             {title},
		"description":       {fmt.Sprintf("Published via HydraCast from %s", item.SourceName)},
	}
	if _, err := fbPostForm(ctx, client, endpoint, finishForm); err != nil {
		return &PublishResult{Status: "failed", Error: fmt.Errorf("upload finish: %w", err)}, nil
	}

	return &PublishResult{
		RemoteID:  videoID,
		RemoteURL: fmt.Sprintf("https://www.facebook.com/%s/videos/%s", f.cfg.PageID, videoID),
		Status:    "published",
	}, nil
}

type fbResponse struct {
	UploadSessionID string `json:"upload_session_id"`
	VideoID         string `json:"video_id"`
	StartOffset     string `json:"start_offset"`
	Error           *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    int    `json:"code"`
	} `json:"error"`
}

func fbPostForm(ctx context.Context, client *http.Client, endpoint string, form url.Values) (*fbResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return fbSend(ctx, client, req)
}

func fbSend(ctx context.Context, client *http.Client, req *http.Request) (*fbResponse, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var fr fbResponse
	if jerr := json.Unmarshal(data, &fr); jerr != nil {
		return nil, fmt.Errorf("HTTP %d (undecodable): %s", resp.StatusCode, fbTruncate(string(data)))
	}
	if fr.Error != nil {
		return nil, fmt.Errorf("HTTP %d: %s (code %d)", resp.StatusCode, fr.Error.Message, fr.Error.Code)
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, fbTruncate(string(data)))
	}
	return &fr, nil
}

func fbTruncate(s string) string {
	if len(s) <= 512 {
		return s
	}
	return s[:512] + "..."
}
