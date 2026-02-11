# Suno API Client

This Go client provides a wrapper around the third-party [suno-api](https://github.com/gcui-art/suno-api) service, which enables API access to Suno.ai's music generation capabilities.

## Important Note

**Suno.ai does not have an official API.** This implementation uses the third-party `suno-api` project, which runs a Node.js server that interfaces with Suno.ai using browser automation.

## Prerequisites

### 1. Set up suno-api Server

You need to have the suno-api server running. It's recommended to run it on the same VPS as your application.

#### Installation

```bash
git clone https://github.com/gcui-art/suno-api.git
cd suno-api
npm install
```

#### Configuration

Create a `.env` file in the suno-api directory with the following required variables:

```env
# Your Suno.ai account cookie (see instructions below)
SUNO_COOKIE=<your-cookie-value>

# 2Captcha API key for solving hCaptcha challenges
TWOCAPTCHA_KEY=<your-2captcha-api-key>

# Browser configuration
BROWSER=chromium
BROWSER_GHOST_CURSOR=false
BROWSER_LOCALE=en
BROWSER_HEADLESS=true
```

#### Getting Your Suno Cookie

1. Go to [suno.ai/create](https://suno.ai/create) in your browser
2. Open Developer Tools (F12)
3. Go to the Network tab
4. Refresh the page
5. Find a request containing `?__clerk_api_version`
6. Click on it, go to the Headers tab
7. Find the `Cookie` section and copy the entire cookie value

#### Getting 2Captcha API Key

1. Sign up at [2captcha.com](https://2captcha.com) (or [rucaptcha.com](https://rucaptcha.com) if in Russia/Belarus)
2. Top up your balance
3. Get your API key from the dashboard

**Note:** 2Captcha is a paid service that uses real workers to solve CAPTCHAs. Running on macOS typically results in fewer CAPTCHAs.

#### Start the suno-api Server

```bash
npm run dev
```

By default, the server runs on `http://localhost:3000`. Test it:

```bash
curl http://localhost:3000/api/get_limit
```

Expected response:
```json
{
  "credits_left": 50,
  "period": "day",
  "monthly_limit": 50,
  "monthly_usage": 0
}
```

### 2. Use the Go Client

#### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "your-project/lib/suno"
)

func main() {
    // Create client pointing to your suno-api server
    client := suno.NewClient("http://localhost:3000")
    
    ctx := context.Background()

    // Check quota
    quota, err := client.GetQuota(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Credits left: %d\n", quota.CreditsLeft)

    // Generate music with a simple prompt
    req := &suno.GenerateRequest{
        Prompt:           "A heavy metal song about coding",
        MakeInstrumental: false,
        WaitAudio:        false, // Don't wait, return immediately with IDs
    }

    responses, err := client.Generate(ctx, req)
    if err != nil {
        log.Fatal(err)
    }

    // Typically returns 2 variations
    for _, resp := range responses {
        fmt.Printf("Generated audio ID: %s\n", resp.ID)
        
        // Wait for completion
        audio, err := client.WaitForCompletion(ctx, resp.ID, 5*time.Second, 60)
        if err != nil {
            log.Printf("Error waiting for %s: %v\n", resp.ID, err)
            continue
        }
        
        fmt.Printf("Audio ready: %s\n", audio.AudioURL)
        fmt.Printf("Title: %s\n", audio.Title)
    }
}
```

#### Custom Generation with Full Control

```go
// Generate with custom lyrics, style, and title
req := &suno.CustomGenerateRequest{
    Prompt:           "Verse 1:\nIn the darkness of the night...",
    Tags:             "heavy metal, epic, dramatic",
    NegativeTags:     "female, pop, acoustic",
    Title:            "Night Warriors",
    MakeInstrumental: false,
    Model:            "chirp-v3-5",
    WaitAudio:        false,
}

responses, err := client.CustomGenerate(ctx, req)
if err != nil {
    log.Fatal(err)
}
```

#### Generate Lyrics Only

```go
// Generate lyrics without creating audio
req := &suno.GenerateLyricsRequest{
    Prompt: "A song about overcoming challenges",
}

lyrics, err := client.GenerateLyrics(ctx, req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Title: %s\n", lyrics.Title)
fmt.Printf("Lyrics:\n%s\n", lyrics.Text)
```

#### Extend Audio Length

```go
// Extend an existing audio clip
req := &suno.ExtendAudioRequest{
    AudioID:    "e76498dc-6ab4-4a10-a19f-8a095790e28d",
    Prompt:     "[lrc]Additional verse here...[endlrc]",
    ContinueAt: "00:30", // Continue from 30 seconds
    Tags:       "rock, energetic",
}

extended, err := client.ExtendAudio(ctx, req)
if err != nil {
    log.Fatal(err)
}
```

#### Generate Stem Tracks

```go
// Separate vocals and instrumentals
req := &suno.GenerateStemsRequest{
    AudioID: "e76498dc-6ab4-4a10-a19f-8a095790e28d",
}

stems, err := client.GenerateStems(ctx, req)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Stems ready: %s\n", stems.AudioURL)
```

#### Get Audio with Pagination

```go
// Get all audios, page 1
audios, err := client.Get(ctx, "", 1)
if err != nil {
    log.Fatal(err)
}

for _, audio := range audios {
    fmt.Printf("ID: %s, Title: %s\n", audio.ID, audio.Title)
}
```

#### Get Persona Information

```go
// Get persona details
persona, err := client.GetPersona(ctx, "persona-id-here", 1)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Persona: %s\n", persona.Persona.Name)
fmt.Printf("Total Clips: %d\n", persona.TotalResults)
```

## API Reference

### Client Methods

#### `NewClient(baseURL string) *Client`
Creates a new Suno API client. The `baseURL` should point to your running suno-api server.

#### `Generate(ctx context.Context, req *GenerateRequest) ([]AudioInfo, error)`
Generates music using a simple text prompt. Automatically fills in lyrics. Returns typically 2 variations. Consumes 10 credits.

#### `CustomGenerate(ctx context.Context, req *CustomGenerateRequest) ([]AudioInfo, error)`
Generates music with full control over lyrics, style, title, and tags. Returns typically 2 variations. Consumes 10 credits.

#### `ExtendAudio(ctx context.Context, req *ExtendAudioRequest) ([]AudioInfo, error)`
Extends the length of an existing audio clip. Can continue from a specific timestamp.

#### `GenerateStems(ctx context.Context, req *GenerateStemsRequest) (*AudioInfo, error)`
Generates stem tracks (separate audio and music tracks) from an existing audio clip.

#### `GenerateLyrics(ctx context.Context, req *GenerateLyricsRequest) (*LyricsResponse, error)`
Generates lyrics based on a text prompt without creating audio.

#### `Concat(ctx context.Context, req *ConcatRequest) (*AudioInfo, error)`
Generates the whole song from audio extensions.

#### `Get(ctx context.Context, ids string, page int) ([]AudioInfo, error)`
Retrieves audio information by ID(s). Pass comma-separated IDs or empty string for all. Use `page` for pagination (0 = no pagination).

#### `GetClip(ctx context.Context, id string) (*AudioInfo, error)`
Retrieves detailed clip information by ID.

#### `GetAlignedLyrics(ctx context.Context, songID string) (*AudioInfo, error)`
Gets lyric alignment with timestamps for a song.

#### `GetPersona(ctx context.Context, id string, page int) (*PersonaResponse, error)`
Retrieves persona information including associated clips and metadata.

#### `GetQuota(ctx context.Context) (*QuotaInfo, error)`
Gets current account quota and usage information.

#### `WaitForCompletion(ctx context.Context, id string, pollInterval time.Duration, maxRetries int) (*AudioInfo, error)`
Polls the API until audio generation is complete.

### Types

#### `GenerateRequest`
```go
type GenerateRequest struct {
    Prompt           string // Description of the music to generate
    MakeInstrumental bool   // Generate instrumental version (no vocals)
    Model            string // Model name: "chirp-v3-5" (default) or "chirp-v3-0"
    WaitAudio        bool   // Wait for audio to be ready before returning
}
```

#### `CustomGenerateRequest`
```go
type CustomGenerateRequest struct {
    Prompt           string // Lyrics or detailed description
    Tags             string // Music style/genre (e.g., "rock, energetic")
    NegativeTags     string // Tags to avoid (e.g., "female, edm")
    Title            string // Song title
    MakeInstrumental bool   // Generate instrumental version
    Model            string // Model name: "chirp-v3-5" (default) or "chirp-v3-0"
    WaitAudio        bool   // Wait for audio to be ready
}
```

#### `ExtendAudioRequest`
```go
type ExtendAudioRequest struct {
    AudioID      string // ID of the audio clip to extend
    Prompt       string // Additional lyrics (optional)
    ContinueAt   string // Extend from mm:ss, e.g., "00:30" (optional)
    Title        string // Title for the extended part (optional)
    Tags         string // Music style/genre (optional)
    NegativeTags string // Tags to avoid (optional)
    Model        string // Model name (optional)
}
```

#### `GenerateStemsRequest`
```go
type GenerateStemsRequest struct {
    AudioID string // ID of the song to generate stems for
}
```

#### `GenerateLyricsRequest`
```go
type GenerateLyricsRequest struct {
    Prompt string // Description of what the lyrics should be about
}
```

#### `ConcatRequest`
```go
type ConcatRequest struct {
    ClipID string // Clip ID to concatenate
}
```

#### `AudioInfo` (alias: `GenerateResponse`)
Contains full information about generated audio:
- `ID` - Unique identifier
- `Title` - Song title
- `ImageURL` - Cover image URL
- `Lyric` - Song lyrics
- `AudioURL` - URL to download the audio file
- `VideoURL` - URL to video version
- `CreatedAt` - Creation timestamp
- `ModelName` - Model used (e.g., "chirp-v3-5")
- `Status` - Generation status: "submitted", "queue", "streaming", "complete"
- `GPTDescriptionPrompt` - Original user prompt (simple mode)
- `Prompt` - Final prompt used for generation
- `Type` - Type of audio
- `Tags` - Music genre/style tags
- `Duration` - Length in seconds

#### `LyricsResponse`
```go
type LyricsResponse struct {
    Text   string // Generated lyrics
    Title  string // Generated title
    Status string // Generation status
}
```

#### `PersonaResponse`
```go
type PersonaResponse struct {
    Persona      Persona // Persona details
    TotalResults int     // Total number of clips
    CurrentPage  int     // Current page number
    IsFollowing  bool    // Whether user follows this persona
}
```

#### `QuotaInfo`
```go
type QuotaInfo struct {
    CreditsLeft   int    // Remaining credits (each song costs 5 credits)
    Period        string // Quota period ("day", "month")
    MonthlyLimit  int    // Monthly credit limit
    MonthlyUsage  int    // Credits used this month
}
```

## Production Deployment

### Running suno-api as a Service

Create a systemd service file `/etc/systemd/system/suno-api.service`:

```ini
[Unit]
Description=Suno API Service
After=network.target

[Service]
Type=simple
User=youruser
WorkingDirectory=/path/to/suno-api
Environment=NODE_ENV=production
ExecStart=/usr/bin/npm start
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl enable suno-api
sudo systemctl start suno-api
sudo systemctl status suno-api
```

### Security Considerations

- **Never expose the suno-api server to the public internet** - it should only be accessible from localhost or your private network
- Keep your `SUNO_COOKIE` secure and rotate it periodically
- Monitor your 2Captcha spending to avoid unexpected costs
- Consider rate limiting your Go application to avoid excessive API calls

## Troubleshooting

### CAPTCHAs Too Frequent

- Use macOS if possible (gets fewer CAPTCHAs than Linux/Windows)
- Ensure `BROWSER_HEADLESS=true` is set
- Check your 2Captcha balance
- Try different `BROWSER_LOCALE` values

### Cookie Expired

If you get authentication errors:
1. Get a fresh cookie from suno.ai (follow instructions above)
2. Update the `SUNO_COOKIE` environment variable
3. Restart the suno-api server

### Generation Taking Too Long

- Suno can take 1-3 minutes per song
- Use `WaitAudio: false` and poll with `Get()` instead
- Consider implementing a queue system for multiple requests

## License

This client wrapper is part of the workflower project. The underlying suno-api is licensed under LGPL-3.0.
