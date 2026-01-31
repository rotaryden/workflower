package templates

import (
	"html/template"
)

// BaseLayout returns the common HTML layout with Tailwind CSS
const baseLayout = `<!DOCTYPE html>
<html lang="en" class="h-full">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - Suno Workflow</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <link href="https://fonts.googleapis.com/css2?family=Playfair+Display:wght@400;600;700&family=JetBrains+Mono:wght@400;500&family=Outfit:wght@300;400;500;600&display=swap" rel="stylesheet">
    <style>
        :root {
            --gradient-start: #0f0c29;
            --gradient-mid: #302b63;
            --gradient-end: #24243e;
            --accent-gold: #fbbf24;
            --accent-rose: #f43f5e;
            --accent-violet: #8b5cf6;
        }
        body {
            font-family: 'Outfit', sans-serif;
            background: linear-gradient(135deg, var(--gradient-start) 0%, var(--gradient-mid) 50%, var(--gradient-end) 100%);
            min-height: 100vh;
        }
        .font-display {
            font-family: 'Playfair Display', serif;
        }
        .font-mono {
            font-family: 'JetBrains Mono', monospace;
        }
        .glass-card {
            background: rgba(255, 255, 255, 0.08);
            backdrop-filter: blur(20px);
            border: 1px solid rgba(255, 255, 255, 0.1);
        }
        .glow-border {
            box-shadow: 0 0 30px rgba(139, 92, 246, 0.3), inset 0 0 30px rgba(139, 92, 246, 0.05);
        }
        .btn-primary {
            background: linear-gradient(135deg, var(--accent-violet) 0%, var(--accent-rose) 100%);
            transition: all 0.3s ease;
        }
        .btn-primary:hover {
            transform: translateY(-2px);
            box-shadow: 0 10px 40px rgba(139, 92, 246, 0.4);
        }
        .input-glow:focus {
            box-shadow: 0 0 20px rgba(139, 92, 246, 0.4);
            border-color: var(--accent-violet);
        }
        .animate-float {
            animation: float 6s ease-in-out infinite;
        }
        @keyframes float {
            0%, 100% { transform: translateY(0px); }
            50% { transform: translateY(-10px); }
        }
        .bg-pattern {
            background-image: 
                radial-gradient(circle at 20% 80%, rgba(139, 92, 246, 0.1) 0%, transparent 50%),
                radial-gradient(circle at 80% 20%, rgba(244, 63, 94, 0.1) 0%, transparent 50%),
                radial-gradient(circle at 40% 40%, rgba(251, 191, 36, 0.05) 0%, transparent 30%);
        }
        textarea::-webkit-scrollbar {
            width: 8px;
        }
        textarea::-webkit-scrollbar-track {
            background: rgba(255, 255, 255, 0.05);
            border-radius: 4px;
        }
        textarea::-webkit-scrollbar-thumb {
            background: rgba(139, 92, 246, 0.5);
            border-radius: 4px;
        }
    </style>
</head>
<body class="bg-pattern text-white antialiased">
    <div class="min-h-screen flex flex-col">
        <!-- Header -->
        <header class="py-6 px-8">
            <nav class="max-w-6xl mx-auto flex items-center justify-between">
                <a href="/" class="flex items-center gap-3 group">
                    <div class="w-12 h-12 rounded-xl bg-gradient-to-br from-violet-500 to-rose-500 flex items-center justify-center animate-float">
                        <svg class="w-7 h-7 text-white" fill="currentColor" viewBox="0 0 24 24">
                            <path d="M12 3v10.55c-.59-.34-1.27-.55-2-.55-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4V7h4V3h-6z"/>
                        </svg>
                    </div>
                    <span class="font-display text-2xl font-semibold tracking-wide">Suno<span class="text-violet-400">Flow</span></span>
                </a>
                <div class="flex items-center gap-4">
                    <a href="/" class="px-4 py-2 text-gray-300 hover:text-white transition">Home</a>
                    <a href="/workflows" class="px-4 py-2 text-gray-300 hover:text-white transition">Workflows</a>
                </div>
            </nav>
        </header>
        
        <!-- Main Content -->
        <main class="flex-1 px-8 py-12">
            <div class="max-w-4xl mx-auto">
                {{template "content" .}}
            </div>
        </main>
        
        <!-- Footer -->
        <footer class="py-6 px-8 text-center text-gray-500 text-sm">
            <p>Powered by AI • Built with Go & Tailwind</p>
        </footer>
    </div>
</body>
</html>`

// StartPageContent is the workflow starter form
const startPageContent = `{{define "content"}}
<div class="text-center mb-12">
    <h1 class="font-display text-5xl font-bold mb-4 bg-gradient-to-r from-violet-400 via-rose-400 to-amber-400 bg-clip-text text-transparent">
        Create Your Song
    </h1>
    <p class="text-gray-400 text-lg max-w-2xl mx-auto">
        Transform your ideas into music. Describe your vision and let AI craft the perfect lyrics and sound.
    </p>
</div>

<form action="/workflow/start" method="POST" enctype="multipart/form-data" class="space-y-8">
    <div class="glass-card glow-border rounded-2xl p-8 space-y-6">
        <!-- Task Description -->
        <div>
            <label for="task_description" class="block text-sm font-medium text-gray-300 mb-2">
                <span class="flex items-center gap-2">
                    <svg class="w-5 h-5 text-violet-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"/>
                    </svg>
                    Song Description
                </span>
            </label>
            <textarea 
                name="task_description" 
                id="task_description" 
                rows="6" 
                required
                placeholder="Describe what you want your song to be about. Include emotions, themes, story elements, or any specific ideas you want to capture..."
                class="w-full px-5 py-4 bg-white/5 border border-white/10 rounded-xl text-white placeholder-gray-500 focus:outline-none input-glow transition resize-none"
            ></textarea>
        </div>

        <!-- Premium Toggle -->
        <div class="flex items-center justify-between p-4 bg-gradient-to-r from-amber-500/10 to-rose-500/10 rounded-xl border border-amber-500/20">
            <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-gradient-to-br from-amber-400 to-rose-500 flex items-center justify-center">
                    <svg class="w-5 h-5 text-white" fill="currentColor" viewBox="0 0 24 24">
                        <path d="M12 2L15.09 8.26L22 9.27L17 14.14L18.18 21.02L12 17.77L5.82 21.02L7 14.14L2 9.27L8.91 8.26L12 2Z"/>
                    </svg>
                </div>
                <div>
                    <p class="font-medium text-white">Premium Features</p>
                    <p class="text-sm text-gray-400">Enable Persona & Inspo for enhanced generation</p>
                </div>
            </div>
            <label class="relative inline-flex items-center cursor-pointer">
                <input type="checkbox" name="is_premium" value="true" class="sr-only peer">
                <div class="w-14 h-7 bg-gray-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-0.5 after:left-[4px] after:bg-white after:rounded-full after:h-6 after:w-6 after:transition-all peer-checked:bg-gradient-to-r peer-checked:from-amber-400 peer-checked:to-rose-500"></div>
            </label>
        </div>

        <!-- Audio Upload -->
        <div>
            <label class="block text-sm font-medium text-gray-300 mb-2">
                <span class="flex items-center gap-2">
                    <svg class="w-5 h-5 text-rose-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3"/>
                    </svg>
                    Audio Reference (Optional)
                </span>
            </label>
            <div class="relative">
                <input 
                    type="file" 
                    name="audio_file" 
                    id="audio_file" 
                    accept="audio/*"
                    class="hidden"
                    onchange="updateFileName(this)"
                >
                <label for="audio_file" class="flex items-center justify-center gap-3 px-6 py-8 bg-white/5 border-2 border-dashed border-white/20 rounded-xl cursor-pointer hover:border-violet-500/50 hover:bg-violet-500/5 transition group">
                    <svg class="w-8 h-8 text-gray-500 group-hover:text-violet-400 transition" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"/>
                    </svg>
                    <span id="file-name" class="text-gray-400 group-hover:text-gray-300 transition">
                        Drop an audio file or click to browse
                    </span>
                </label>
            </div>
        </div>
    </div>

    <!-- Submit Button -->
    <div class="flex justify-center">
        <button type="submit" class="btn-primary px-12 py-4 rounded-xl text-lg font-semibold text-white flex items-center gap-3">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/>
            </svg>
            Start Workflow
        </button>
    </div>
</form>

<script>
function updateFileName(input) {
    const fileName = input.files[0]?.name || 'Drop an audio file or click to browse';
    document.getElementById('file-name').textContent = fileName;
}
</script>
{{end}}`

// ReviewPageContent is the human-in-the-loop review page
const reviewPageContent = `{{define "content"}}
<div class="text-center mb-10">
    <div class="inline-flex items-center gap-2 px-4 py-2 bg-amber-500/20 rounded-full text-amber-400 text-sm font-medium mb-4">
        <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
            <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
        </svg>
        Awaiting Your Review
    </div>
    <h1 class="font-display text-4xl font-bold mb-3 text-white">Review & Approve</h1>
    <p class="text-gray-400 max-w-xl mx-auto">
        Review the generated content below. Edit as needed, then approve to send to Suno.
    </p>
</div>

<form action="/workflow/{{.Workflow.ID}}/submit" method="POST" class="space-y-6">
    <!-- Original Description -->
    <div class="glass-card rounded-xl p-6">
        <h3 class="flex items-center gap-2 text-sm font-medium text-gray-400 mb-3">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
            </svg>
            Original Description
        </h3>
        <p class="text-gray-300 leading-relaxed">{{.Workflow.TaskDescription}}</p>
    </div>

    <!-- Lyrics Editor -->
    <div class="glass-card glow-border rounded-xl p-6">
        <label class="flex items-center gap-2 text-lg font-semibold text-white mb-4">
            <svg class="w-5 h-5 text-violet-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2z"/>
            </svg>
            Lyrics with Instructions
        </label>
        <textarea 
            name="edited_lyrics" 
            rows="16" 
            class="w-full px-4 py-4 bg-black/30 border border-white/10 rounded-lg text-white font-mono text-sm focus:outline-none input-glow transition resize-none leading-relaxed"
        >{{.Workflow.EditedLyrics}}</textarea>
    </div>

    <!-- Properties -->
    <div class="grid md:grid-cols-2 gap-6">
        <!-- Style -->
        <div class="glass-card rounded-xl p-5">
            <label class="block text-sm font-medium text-gray-300 mb-2">Style</label>
            <input 
                type="text" 
                name="style" 
                value="{{.Workflow.EditedProperties.Style}}"
                class="w-full px-4 py-3 bg-white/5 border border-white/10 rounded-lg text-white focus:outline-none input-glow transition"
            >
        </div>
        
        <!-- Vocal Type -->
        <div class="glass-card rounded-xl p-5">
            <label class="block text-sm font-medium text-gray-300 mb-2">Vocal Type</label>
            <input 
                type="text" 
                name="vocal_type" 
                value="{{.Workflow.EditedProperties.VocalType}}"
                class="w-full px-4 py-3 bg-white/5 border border-white/10 rounded-lg text-white focus:outline-none input-glow transition"
            >
        </div>
        
        <!-- Weirdness -->
        <div class="glass-card rounded-xl p-5">
            <label class="block text-sm font-medium text-gray-300 mb-2">
                Weirdness: <span id="weirdness-value">{{printf "%.1f" .Workflow.EditedProperties.Weirdness}}</span>
            </label>
            <input 
                type="range" 
                name="weirdness" 
                min="0" 
                max="1" 
                step="0.1" 
                value="{{.Workflow.EditedProperties.Weirdness}}"
                oninput="document.getElementById('weirdness-value').textContent = parseFloat(this.value).toFixed(1)"
                class="w-full h-2 bg-gray-700 rounded-lg appearance-none cursor-pointer accent-violet-500"
            >
        </div>
        
        <!-- Style Influence -->
        <div class="glass-card rounded-xl p-5">
            <label class="block text-sm font-medium text-gray-300 mb-2">Style Influence</label>
            <input 
                type="text" 
                name="style_influence" 
                value="{{.Workflow.EditedProperties.StyleInfluence}}"
                class="w-full px-4 py-3 bg-white/5 border border-white/10 rounded-lg text-white focus:outline-none input-glow transition"
            >
        </div>
    </div>

    {{if .Workflow.IsPremium}}
    <!-- Premium Features -->
    <div class="glass-card rounded-xl p-6 border border-amber-500/30">
        <h3 class="flex items-center gap-2 text-lg font-semibold text-amber-400 mb-4">
            <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                <path d="M12 2L15.09 8.26L22 9.27L17 14.14L18.18 21.02L12 17.77L5.82 21.02L7 14.14L2 9.27L8.91 8.26L12 2Z"/>
            </svg>
            Premium Features
        </h3>
        <div class="grid md:grid-cols-2 gap-4">
            <div>
                <label class="block text-sm font-medium text-gray-300 mb-2">Persona</label>
                <textarea 
                    name="persona" 
                    rows="3"
                    class="w-full px-4 py-3 bg-white/5 border border-white/10 rounded-lg text-white focus:outline-none input-glow transition resize-none text-sm"
                >{{if .Workflow.PersonaInspo}}{{.Workflow.PersonaInspo.Persona}}{{end}}</textarea>
            </div>
            <div>
                <label class="block text-sm font-medium text-gray-300 mb-2">Inspo</label>
                <textarea 
                    name="inspo" 
                    rows="3"
                    class="w-full px-4 py-3 bg-white/5 border border-white/10 rounded-lg text-white focus:outline-none input-glow transition resize-none text-sm"
                >{{if .Workflow.PersonaInspo}}{{.Workflow.PersonaInspo.Inspo}}{{end}}</textarea>
            </div>
        </div>
    </div>
    {{end}}

    {{if .Workflow.AudioFileName}}
    <!-- Audio Reference -->
    <div class="glass-card rounded-xl p-5 flex items-center gap-4">
        <div class="w-12 h-12 rounded-lg bg-rose-500/20 flex items-center justify-center">
            <svg class="w-6 h-6 text-rose-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2z"/>
            </svg>
        </div>
        <div>
            <p class="text-sm text-gray-400">Audio Reference</p>
            <p class="text-white font-medium">{{.Workflow.AudioFileName}}</p>
        </div>
    </div>
    {{end}}

    <!-- Action Buttons -->
    <div class="flex flex-col sm:flex-row gap-4 justify-center pt-4">
        <button 
            type="submit" 
            name="action" 
            value="approve"
            class="btn-primary px-10 py-4 rounded-xl text-lg font-semibold text-white flex items-center justify-center gap-2"
        >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
            </svg>
            Approve & Send to Suno
        </button>
        <button 
            type="submit" 
            name="action" 
            value="reject"
            class="px-10 py-4 rounded-xl text-lg font-medium text-gray-400 border border-gray-600 hover:border-rose-500 hover:text-rose-400 transition flex items-center justify-center gap-2"
        >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
            Reject
        </button>
    </div>
</form>
{{end}}`

// StatusPageContent shows workflow status
const statusPageContent = `{{define "content"}}
<div class="text-center">
    <div class="inline-flex items-center justify-center w-20 h-20 rounded-full {{if eq .Workflow.Status "completed"}}bg-green-500/20{{else if eq .Workflow.Status "failed"}}bg-rose-500/20{{else if eq .Workflow.Status "rejected"}}bg-gray-500/20{{else}}bg-violet-500/20{{end}} mb-6">
        {{if eq .Workflow.Status "completed"}}
        <svg class="w-10 h-10 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
        </svg>
        {{else if eq .Workflow.Status "failed"}}
        <svg class="w-10 h-10 text-rose-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
        </svg>
        {{else if eq .Workflow.Status "rejected"}}
        <svg class="w-10 h-10 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
        </svg>
        {{else}}
        <svg class="w-10 h-10 text-violet-400 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
        {{end}}
    </div>
    
    <h1 class="font-display text-4xl font-bold mb-3 text-white">
        {{if eq .Workflow.Status "completed"}}Song Created!{{else if eq .Workflow.Status "failed"}}Generation Failed{{else if eq .Workflow.Status "rejected"}}Workflow Rejected{{else if eq .Workflow.Status "processing"}}Processing...{{else if eq .Workflow.Status "awaiting_review"}}Awaiting Review{{else}}{{.Workflow.Status}}{{end}}
    </h1>
    
    <p class="text-gray-400 mb-8">Workflow ID: <span class="font-mono text-violet-400">{{.Workflow.ID}}</span></p>

    <div class="glass-card rounded-xl p-6 text-left max-w-2xl mx-auto space-y-4">
        <div class="flex justify-between py-3 border-b border-white/10">
            <span class="text-gray-400">Status</span>
            <span class="{{if eq .Workflow.Status "completed"}}text-green-400{{else if eq .Workflow.Status "failed"}}text-rose-400{{else}}text-violet-400{{end}} font-medium capitalize">{{.Workflow.Status}}</span>
        </div>
        <div class="flex justify-between py-3 border-b border-white/10">
            <span class="text-gray-400">Created</span>
            <span class="text-white">{{.Workflow.CreatedAt.Format "Jan 02, 2006 15:04"}}</span>
        </div>
        {{if .Workflow.SunoJobID}}
        <div class="flex justify-between py-3 border-b border-white/10">
            <span class="text-gray-400">Suno Job ID</span>
            <span class="text-white font-mono">{{.Workflow.SunoJobID}}</span>
        </div>
        {{end}}
        {{if .Workflow.ErrorMsg}}
        <div class="py-3">
            <span class="text-gray-400 block mb-2">Error</span>
            <p class="text-rose-400 bg-rose-500/10 px-4 py-3 rounded-lg text-sm">{{.Workflow.ErrorMsg}}</p>
        </div>
        {{end}}
    </div>

    <div class="mt-8">
        <a href="/" class="inline-flex items-center gap-2 text-violet-400 hover:text-violet-300 transition">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"/>
            </svg>
            Start New Workflow
        </a>
    </div>
</div>
{{end}}`

// WorkflowsListContent shows all workflows
const workflowsListContent = `{{define "content"}}
<div class="text-center mb-10">
    <h1 class="font-display text-4xl font-bold mb-3 text-white">Your Workflows</h1>
    <p class="text-gray-400">Track and manage all your song generation workflows</p>
</div>

{{if .Workflows}}
<div class="space-y-4">
    {{range .Workflows}}
    <a href="/workflow/{{.ID}}" class="block glass-card rounded-xl p-5 hover:border-violet-500/50 transition group">
        <div class="flex items-center justify-between">
            <div class="flex-1 min-w-0">
                <p class="text-white font-medium truncate group-hover:text-violet-300 transition">
                    {{if gt (len .TaskDescription) 60}}{{slice .TaskDescription 0 60}}...{{else}}{{.TaskDescription}}{{end}}
                </p>
                <p class="text-sm text-gray-500 mt-1">
                    {{.CreatedAt.Format "Jan 02, 2006 15:04"}}
                </p>
            </div>
            <div class="flex items-center gap-4 ml-4">
                <span class="px-3 py-1 rounded-full text-xs font-medium
                    {{if eq .Status "completed"}}bg-green-500/20 text-green-400
                    {{else if eq .Status "failed"}}bg-rose-500/20 text-rose-400
                    {{else if eq .Status "rejected"}}bg-gray-500/20 text-gray-400
                    {{else if eq .Status "awaiting_review"}}bg-amber-500/20 text-amber-400
                    {{else}}bg-violet-500/20 text-violet-400{{end}}
                ">
                    {{.Status}}
                </span>
                <svg class="w-5 h-5 text-gray-600 group-hover:text-violet-400 transition" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
                </svg>
            </div>
        </div>
    </a>
    {{end}}
</div>
{{else}}
<div class="text-center py-16">
    <div class="w-16 h-16 rounded-full bg-gray-800 flex items-center justify-center mx-auto mb-4">
        <svg class="w-8 h-8 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2z"/>
        </svg>
    </div>
    <p class="text-gray-500 mb-4">No workflows yet</p>
    <a href="/" class="inline-flex items-center gap-2 text-violet-400 hover:text-violet-300 transition">
        Create your first song →
    </a>
</div>
{{end}}
{{end}}`

// Template instances
var (
	StartPage     *template.Template
	ReviewPage    *template.Template
	StatusPage    *template.Template
	WorkflowsList *template.Template
)

// Init initializes all templates
func Init() error {
	var err error
	
	StartPage, err = template.New("start").Parse(baseLayout + startPageContent)
	if err != nil {
		return err
	}
	
	ReviewPage, err = template.New("review").Parse(baseLayout + reviewPageContent)
	if err != nil {
		return err
	}
	
	StatusPage, err = template.New("status").Parse(baseLayout + statusPageContent)
	if err != nil {
		return err
	}
	
	WorkflowsList, err = template.New("list").Parse(baseLayout + workflowsListContent)
	if err != nil {
		return err
	}
	
	return nil
}

// PageData represents the data passed to templates
type PageData struct {
	Title    string
	Workflow interface{}
	Workflows interface{}
}

