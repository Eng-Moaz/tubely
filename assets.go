package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

)


type ffprobeOutput struct {
	Streams []struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"streams"`
}

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func getAssetPath(mediaType string) string {
	base := make([]byte, 32)
	_, err := rand.Read(base)
	if err != nil {
		panic("failed to generate random bytes")
	}
	id := base64.RawURLEncoding.EncodeToString(base)

	ext := mediaTypeToExt(mediaType)
	return fmt.Sprintf("%s%s", id, ext)
}

func mediaTypeToExt(mediaType string) string {
	parts := strings.Split(mediaType, "/")
	if len(parts) != 2 {
		return ".bin"
	}
	return "." + parts[1]
}

func (cfg apiConfig) getObjectURL(key string) string {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, key)
}

func getVideoAspectRatio(filePath string) (string, error){
	command := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	buf := make([]byte, 0)
	buffer := bytes.NewBuffer(buf)
	command.Stdout = buffer
	err := command.Run()

	if err != nil{
		return "", fmt.Errorf("Failed to execute command")
	}

	var data ffprobeOutput

	err = json.Unmarshal(buffer.Bytes(), &data)
	if err != nil{
		return "", fmt.Errorf("Failed to Unmarshal")
	}

	width, height :=data.Streams[0].Width, data.Streams[0].Height
	r := float64(width) / float64(height)

	if isCloseTo(r, float64(16)/float64(9), 0.1){
		return "16:9", nil
	}else if isCloseTo(r, float64(9)/float64(16), 0.1){
		return "9:16", nil
	}else{
		return "other", nil
	}
}

func isCloseTo(value, target, tolerance float64) bool {
	diff := value - target
	if diff < 0 {
		diff = -diff // absolute value
	}
	return diff <= tolerance
}


func processVideoForFastStart(filePath string) (string, error){
	outputPath := fmt.Sprint(filePath, ".processing")
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputPath)
	if err := cmd.Run() ; err != nil {
		return filePath, fmt.Errorf("Failed to execute command")
	}
	return outputPath, nil
}
