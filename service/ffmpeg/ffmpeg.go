package ffmpeg

import (
	_ "embed"
	"errors"
	"go.uber.org/zap"
	"jiayou_backend_spider/engine"
	"jiayou_backend_spider/service/common"
	"jiayou_backend_spider/utils"
	"os"
	"path/filepath"
)

//go:embed ffmpeg.exe
var ffmpeg []byte

func OnLoad(engine *engine.Engine) error {
	var ffmpegPath = filepath.Join(utils.MustGetDefaultHomeDir("ffmpeg"), common.DefaultFFmpegPath)
	_, err := os.Stat(ffmpegPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err = os.WriteFile(ffmpegPath, ffmpeg, 0644); err != nil {
				common.GLogger.Error("write ffmpeg failed", zap.String("ffmpeg", ffmpegPath), zap.Error(err))
				return err
			}
		} else {
			common.GLogger.Error("stat ffmpeg path failed", zap.String("ffmpeg", ffmpegPath), zap.Error(err))
			return err
		}
	}
	common.GLogger.Info("ffmpeg loaded", zap.String("ffmpeg", ffmpegPath))
	common.DefaultFFmpegPath = ffmpegPath
	return nil
}
