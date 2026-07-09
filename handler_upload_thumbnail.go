package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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


	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse", err)
		return
	}

	mediaType := header.Header.Get("Content-Type")
	if mediaType == ""{
		respondWithError(w, http.StatusBadRequest, "Couldn't parse", err)
		return
	}
	extension := strings.TrimPrefix(mediaType, "image/")

	defer file.Close()

	filename := videoID.String() + "." + extension
	thumbnailPath := filepath.Join(cfg.assetsRoot, filename)
	fileptr, err := os.Create(thumbnailPath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error in retrieving", err)
		return
	}

	defer fileptr.Close()
	_, err = io.Copy(fileptr, file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error in retrieving", err)
		return
	}

	vidMetaData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error in retrieving", err)
		return
	}

	if vidMetaData.UserID != userID{
		respondWithError(w, http.StatusUnauthorized, "Couldn't authorize", nil)
		return
	}

	publicURL := fmt.Sprintf("http://localhost:%s/assets/%s.%s", cfg.port, videoIDString , extension)
	vidMetaData.ThumbnailURL = &publicURL
	err = cfg.db.UpdateVideo(vidMetaData)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse", err)
		return
	}

	respondWithJSON(w, http.StatusOK, vidMetaData)
}
