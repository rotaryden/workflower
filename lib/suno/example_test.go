package suno_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"workflower/lib/suno"
)

// ExampleClient_Generate demonstrates simple music generation using a prompt
func ExampleClient_Generate() {
	client := suno.NewClient("http://localhost:3000")
	ctx := context.Background()

	req := &suno.GenerateRequest{
		Prompt:           "A relaxing jazz piano piece for studying",
		MakeInstrumental: true,
		Model:            "chirp-v3-5",
		WaitAudio:        false,
	}

	audios, err := client.Generate(ctx, req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Generated %d audio variations\n", len(audios))
	for _, audio := range audios {
		fmt.Printf("ID: %s, Status: %s\n", audio.ID, audio.Status)
	}
}

// ExampleClient_CustomGenerate demonstrates custom music generation with full control
func ExampleClient_CustomGenerate() {
	client := suno.NewClient("http://localhost:3000")
	ctx := context.Background()

	req := &suno.CustomGenerateRequest{
		Prompt: `[Verse 1]
In the silence of the night
Stars are shining oh so bright

[Chorus]
We are dancing in the moonlight
Everything will be alright`,
		Tags:             "pop, dreamy, emotional, female vocals",
		NegativeTags:     "metal, aggressive, dark",
		Title:            "Moonlight Dance",
		MakeInstrumental: false,
		Model:            "chirp-v3-5",
		WaitAudio:        false,
	}

	audios, err := client.CustomGenerate(ctx, req)
	if err != nil {
		log.Fatal(err)
	}

	// Wait for completion
	for _, audio := range audios {
		completed, err := client.WaitForCompletion(ctx, audio.ID, 5*time.Second, 60)
		if err != nil {
			log.Printf("Error waiting for %s: %v\n", audio.ID, err)
			continue
		}
		
		fmt.Printf("✅ Song ready!\n")
		fmt.Printf("Title: %s\n", completed.Title)
		fmt.Printf("Audio: %s\n", completed.AudioURL)
		fmt.Printf("Video: %s\n", completed.VideoURL)
	}
}

// ExampleClient_GenerateLyrics demonstrates lyrics generation without audio
func ExampleClient_GenerateLyrics() {
	client := suno.NewClient("http://localhost:3000")
	ctx := context.Background()

	req := &suno.GenerateLyricsRequest{
		Prompt: "A heartfelt song about reunion after years apart",
	}

	lyrics, err := client.GenerateLyrics(ctx, req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Title: %s\n\n", lyrics.Title)
	fmt.Printf("Lyrics:\n%s\n", lyrics.Text)
}

// ExampleClient_ExtendAudio demonstrates extending an existing audio clip
func ExampleClient_ExtendAudio() {
	client := suno.NewClient("http://localhost:3000")
	ctx := context.Background()

	// First, generate a song
	genReq := &suno.GenerateRequest{
		Prompt:    "A short rock intro",
		WaitAudio: false,
	}

	audios, err := client.Generate(ctx, genReq)
	if err != nil {
		log.Fatal(err)
	}

	audioID := audios[0].ID
	
	// Wait for it to complete
	completed, err := client.WaitForCompletion(ctx, audioID, 5*time.Second, 60)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Original song: %s\n", completed.AudioURL)

	// Now extend it
	extendReq := &suno.ExtendAudioRequest{
		AudioID:    audioID,
		Prompt:     "[lrc]Keep rocking all night long[endlrc]",
		ContinueAt: "00:30",
		Tags:       "rock, energetic",
	}

	extended, err := client.ExtendAudio(ctx, extendReq)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Extended version: %s\n", extended[0].ID)
}

// ExampleClient_GenerateStems demonstrates separating audio into stem tracks
func ExampleClient_GenerateStems() {
	client := suno.NewClient("http://localhost:3000")
	ctx := context.Background()

	req := &suno.GenerateStemsRequest{
		AudioID: "your-audio-id-here",
	}

	stems, err := client.GenerateStems(ctx, req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Stems generated: %s\n", stems.AudioURL)
}

// ExampleClient_Get demonstrates retrieving audio information
func ExampleClient_Get() {
	client := suno.NewClient("http://localhost:3000")
	ctx := context.Background()

	// Get specific audios by IDs
	audios, err := client.Get(ctx, "id1,id2,id3", 0)
	if err != nil {
		log.Fatal(err)
	}

	for _, audio := range audios {
		fmt.Printf("ID: %s\n", audio.ID)
		fmt.Printf("Title: %s\n", audio.Title)
		fmt.Printf("Status: %s\n", audio.Status)
		fmt.Printf("Audio URL: %s\n", audio.AudioURL)
		fmt.Println("---")
	}

	// Get all audios with pagination
	allAudios, err := client.Get(ctx, "", 1)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d audios on page 1\n", len(allAudios))
}

// ExampleClient_GetClip demonstrates getting detailed clip information
func ExampleClient_GetClip() {
	client := suno.NewClient("http://localhost:3000")
	ctx := context.Background()

	clip, err := client.GetClip(ctx, "clip-id-here")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Clip: %s\n", clip.Title)
	fmt.Printf("Duration: %.2f seconds\n", clip.Duration)
}

// ExampleClient_GetAlignedLyrics demonstrates getting lyric timing information
func ExampleClient_GetAlignedLyrics() {
	client := suno.NewClient("http://localhost:3000")
	ctx := context.Background()

	lyrics, err := client.GetAlignedLyrics(ctx, "song-id-here")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Aligned lyrics for: %s\n", lyrics.Title)
}

// ExampleClient_Concat demonstrates concatenating audio clips
func ExampleClient_Concat() {
	client := suno.NewClient("http://localhost:3000")
	ctx := context.Background()

	req := &suno.ConcatRequest{
		ClipID: "clip-id-here",
	}

	fullSong, err := client.Concat(ctx, req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Full song: %s\n", fullSong.AudioURL)
}

// ExampleClient_GetPersona demonstrates getting persona information
func ExampleClient_GetPersona() {
	client := suno.NewClient("http://localhost:3000")
	ctx := context.Background()

	persona, err := client.GetPersona(ctx, "persona-id-here", 1)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Persona: %s\n", persona.Persona.Name)
	fmt.Printf("Description: %s\n", persona.Persona.Description)
	fmt.Printf("Total clips: %d\n", persona.TotalResults)
	fmt.Printf("Following: %v\n", persona.IsFollowing)
}

// ExampleClient_GetQuota demonstrates checking account quota
func ExampleClient_GetQuota() {
	client := suno.NewClient("http://localhost:3000")
	ctx := context.Background()

	quota, err := client.GetQuota(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Credits remaining: %d\n", quota.CreditsLeft)
	fmt.Printf("Monthly limit: %d\n", quota.MonthlyLimit)
	fmt.Printf("Monthly usage: %d\n", quota.MonthlyUsage)
	fmt.Printf("Period: %s\n", quota.Period)
}

// ExampleClient_CompleteWorkflow demonstrates a complete music generation workflow
func ExampleClient_CompleteWorkflow() {
	client := suno.NewClient("http://localhost:3000")
	ctx := context.Background()

	// 1. Check quota
	quota, err := client.GetQuota(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Available credits: %d\n", quota.CreditsLeft)

	if quota.CreditsLeft < 10 {
		log.Fatal("Not enough credits")
	}

	// 2. Generate lyrics
	lyricsReq := &suno.GenerateLyricsRequest{
		Prompt: "An uplifting song about new beginnings",
	}

	lyrics, err := client.GenerateLyrics(ctx, lyricsReq)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Generated lyrics: %s\n\n%s\n", lyrics.Title, lyrics.Text)

	// 3. Generate music with custom lyrics
	genReq := &suno.CustomGenerateRequest{
		Prompt:    lyrics.Text,
		Tags:      "uplifting, inspiring, orchestral",
		Title:     lyrics.Title,
		WaitAudio: false,
	}

	audios, err := client.CustomGenerate(ctx, genReq)
	if err != nil {
		log.Fatal(err)
	}

	// 4. Wait for completion and get the first variation
	audio, err := client.WaitForCompletion(ctx, audios[0].ID, 5*time.Second, 60)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✅ Music generated!\n")
	fmt.Printf("Title: %s\n", audio.Title)
	fmt.Printf("Audio: %s\n", audio.AudioURL)
	fmt.Printf("Video: %s\n", audio.VideoURL)
	fmt.Printf("Duration: %.2f seconds\n", audio.Duration)

	// 5. Generate stem tracks (optional)
	stemsReq := &suno.GenerateStemsRequest{
		AudioID: audio.ID,
	}

	stems, err := client.GenerateStems(ctx, stemsReq)
	if err != nil {
		log.Printf("Warning: Could not generate stems: %v\n", err)
	} else {
		fmt.Printf("Stems: %s\n", stems.AudioURL)
	}

	// 6. Check remaining quota
	finalQuota, _ := client.GetQuota(ctx)
	fmt.Printf("Remaining credits: %d\n", finalQuota.CreditsLeft)
}
