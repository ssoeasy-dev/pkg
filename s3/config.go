package s3

// Config содержит настройки подключения к S3-совместимому хранилищу.
type Config struct {
	// Endpoint — URL хранилища. Оставьте пустым для AWS S3.
	// Пример: "https://storage.yandexcloud.net"
	Endpoint string

	// Region — регион хранилища. Например: "ru-central1", "us-east-1".
	Region string

	// AccessKey — access key для аутентификации.
	AccessKey string

	// SecretKey — secret key для аутентификации.
	SecretKey string

	// Bucket — имя бакета, с которым работает клиент.
	Bucket string

	// ForcePathStyle — использовать path-style URLs вместо virtual-hosted-style.
	// Требуется для MinIO и большинства S3-совместимых хранилищ.
	ForcePathStyle bool
}
