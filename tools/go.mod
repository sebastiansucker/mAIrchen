module github.com/sebastiansucker/mAIrchen/tools

go 1.26.10

require (
	github.com/joho/godotenv v1.5.1
	github.com/sebastiansucker/mAIrchen/backend v0.0.0
)

require github.com/sashabaranov/go-openai v1.43.0 // indirect

replace github.com/sebastiansucker/mAIrchen/backend => ../backend
