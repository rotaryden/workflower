<role>
	You are a skilled Golang developer and an AI agentic workflows expert.
</role>
<task_description>
	Create a Golang program running a workflow for creation a song from text input using the Suno service 
</task_description>
<code_design>
	- you should use simple architecture, based on well decomposed functions and modules
	- follow SOLID and DRY principles, SOLID should be very light in object-oriented design, just simple methods on Golang objects and structures.
	- make sure you functional decomposition is highly reusable and maintainable, including future projects with different tasks
	- do NOT create global variables unless it's really neccessary, try to use functions and return values instead
</code_design>
<boilerplate_reuse>
	- treat existing code as boilerplate, the Task implementation should reuse all reasonable parts of this boilerplate.
	- do NOT modify anything in the "lib" folder, just reuse it as library code for the implementation.
	- if you nevertheless think "lib" should be modiffied or extended, ask me first.
</boilerplate_reuse>
<tech_implementation>
	- Workflow should be a Golang program, use Golang version 1.25 or higher
	- it should contain a self-serving HTTP server based on Gin framework, and expose a port to env variable
	- It should render a simple UI HTML templates, created in Go code
	- One HTML page would be a workflow starter with the next fields: task description, is_premium_suno, optional_audio_upload
	- Second HTML UI page should be a human-in-the-loop place before sending to Suno (see workflow details below)
	- use Tailwind for HTML styling, create nice appealing web design
	- NO agentic/LLM frameworks, just direct LLM and tool calls well organized into helper functions
	- Program should use direct calls to ChatGPT models like GPT-5.2 via platform API
	- Program should consist of a plain Golang executable, env file and scripts neccessary for pushing to a remote linux server
	- - NO docker, NO complex setups
	- All neccessary keys and secrets should be in env variables, place ".env_example" with placeholders
</tech_implementation>
<workflow_steps>
	- LLM agentic workflow starts from a user text input and a sound file upload (optional) 
	- text input is an enhanced description about some subject 
	- using LLM (gtp-5.2) with versatile prompt, workflow creates lyrics for this subject suitable for a song 
	- then, using LLM, workflow determines the next Suno properties, suitable for the subject: song style, vocal gender and type or several vocals, lyrics mode, weirdness, style influence 
	- then, using LLM, workflow creates some bracket instructions about voices configuration etc for Suno and places it into lyrics text 
	- there should be a separate step adding Persona and Inspo Suno properties. Tese steps should be turnuble, for premium usage. 
	- there should be a separate step to add Audio example from optional user upload 
	- workflow stops at this stage and enters human-in-the-loop, and notifies user by telegram message (using HTTP call to a TG bot chat) and places a link to the human-in-the-loop HTML page
	- user gets back to human-in-the-loop UI page: it should show workflow results in text form, user should have controls to edit this results if needed and approve or reject. If approved - user edited results wil go further.
	- so if approved: workflow proceeds with posting (modified) lyrics, audio upload and properties to Suno via HTTP API 
</workflow_steps>
<configuration>
	- all potentially configurable parameters should be pushed into linux env variables
</configuration>
<documentation>
	Create concise README file with instruction how to build, run and deploy to linux server
</documentation>
