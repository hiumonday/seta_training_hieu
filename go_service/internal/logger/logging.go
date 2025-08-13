package logger

import (
	"os"

	"github.com/rs/zerolog"
)

var Log zerolog.Logger

// InitLogger khởi tạo logger với output ra cả stdout và file
func InitLogger() {
	// Đảm bảo thư mục logs tồn tại
	logDir := "logs"
	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		panic(err)
	}

	file, err := os.OpenFile("logs/server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	multi := zerolog.MultiLevelWriter(os.Stdout, file)

	// Thiết lập các thông tin mặc định cho mỗi log entry
	Log = zerolog.New(multi).With().Timestamp().Logger()

}

// GetWriter trả về io.Writer để sử dụng với các thư viện khác
// func GetWriter() io.Writer {
// 	return Log
// }
