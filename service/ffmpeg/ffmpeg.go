package ffmpeg

import (
	_ "embed"
	"errors"
	"jiayou_backend_spider/common"
	"jiayou_backend_spider/engine"
	"jiayou_backend_spider/utils"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

//go:embed ffmpeg.exe
var ffmpeg []byte

func OnLoad(engine *engine.Engine) error {
	var logger, err = engine.Log()
	if err != nil {
		return err
	}
	var ffmpegPath = filepath.Join(utils.MustGetDefaultHomeDir("ffmpeg"), common.DefaultFFmpegPath)
	_, err = os.Stat(ffmpegPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err = os.WriteFile(ffmpegPath, ffmpeg, 0644); err != nil {
				logger.Error("write ffmpeg failed", zap.String("ffmpeg", ffmpegPath), zap.Error(err))
				return err
			}
		} else {
			logger.Error("stat ffmpeg path failed", zap.String("ffmpeg", ffmpegPath), zap.Error(err))
			return err
		}
	}
	logger.Info("ffmpeg loaded", zap.String("ffmpeg", ffmpegPath))
	common.DefaultFFmpegPath = ffmpegPath
	return nil
}
