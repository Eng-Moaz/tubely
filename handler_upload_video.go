package main

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	const uploadLimit = 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body, uploadLimit)
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't get video", err)
		return
	}

	if userID != video.UserID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	file, fileHeader, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse", err)
		return
	}
	defer file.Close()

	mediaType := fileHeader.Header.Get("Content-Type")
	if mediaType == ""{
		respondWithError(w, http.StatusBadRequest, "Couldn't parse", err)
		return
	}

	fullMediaType, _, err := mime.ParseMediaType(mediaType)
	if fullMediaType != "video/mp4"{
		respondWithError(w, http.StatusBadRequest, "Invalid mediaType", nil)
		return
	}

	f, err := os.CreateTemp("", "tubely-upload.mp4")
	defer os.Remove(f.Name())
	defer f.Close()
	defer os.Remove(f.Name())

	if fullMediaType != "video/mp4"{
		respondWithError(w, http.StatusBadRequest, "Invalid mediaType", nil)
		return
	}

	_, err = io.Copy(f, file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse", err)
		return
	}

	modifiedPath, err := processVideoForFastStart(f.Name())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse", err)
		return
	}

	modeifiedFile, err := os.Open(modifiedPath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse", err)
		return
	}
	defer modeifiedFile.Close()

	aspectRatio, err := getVideoAspectRatio(f.Name())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse", err)
		return
	}

	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse", err)
		return
	}

	_, err = modeifiedFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse", err)
		return
	}

	objectKey := getAssetPath(fullMediaType)
	var prefix string
	if aspectRatio == "16:9"{
		prefix = "landscape"
	}else if aspectRatio == "9:16"{
		prefix = "portrait"
	}else{
		prefix = "other"
	}
	prefixedObjectKey := fmt.Sprint(prefix, "/", objectKey)
	params := s3.PutObjectInput{
		Bucket: aws.String(cfg.s3Bucket),
		Key: aws.String(prefixedObjectKey),
		Body: modeifiedFile,
		ContentType: aws.String(fullMediaType),
	}
	_, err = cfg.s3Client.PutObject(context.Background(), &params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse", err)
		return
	}

	finalUrl := fmt.Sprint("https://", cfg.s3CfDistribution, "/", prefixedObjectKey)
	video.VideoURL = &finalUrl
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
