module github.com/sebastiansucker/mAIrchen/tools

go 1.21

require (
	github.com/joho/godotenv v1.5.1
	github.com/sebastiansucker/mAIrchen/backend v0.0.0
)

require github.com/sashabaranov/go-openai v1.35.6 // indirect

replace github.com/sebastiansucker/mAIrchen/backend => ../backend
