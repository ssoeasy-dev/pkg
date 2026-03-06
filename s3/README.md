# pkg/s3

Generic S3 клиент для Go-микросервисов SSO Easy. Поверх [aws-sdk-go-v2](https://github.com/aws/aws-sdk-go-v2). Совместим с любым S3-совместимым хранилищем: AWS S3, Tinkoff Cloud Storage, Yandex Object Storage, MinIO.

## Установка

```bash
go get github.com/ssoeasy-dev/pkg/s3@latest
```

## Использование

```go
import s3pkg "github.com/ssoeasy-dev/pkg/s3"

client, err := s3pkg.NewClient(&s3pkg.Config{
    Endpoint:       "https://storage.yandexcloud.net",
    Region:         "ru-central1",
    AccessKey:      os.Getenv("S3_ACCESS_KEY"),
    SecretKey:      os.Getenv("S3_SECRET_KEY"),
    Bucket:         "my-bucket",
    ForcePathStyle: true,
})

// Загрузить объект
err = client.Put(ctx, "videos/abc.mp4", file, "video/mp4")

// Скачать объект с поддержкой Range
out, err := client.Get(ctx, "videos/abc.mp4", "bytes=0-1023")
defer out.Body.Close()

// Получить метаданные
meta, err := client.Head(ctx, "videos/abc.mp4")
fmt.Println(*meta.ContentLength)
```

## API

```go
func NewClient(cfg *Config) (*Client, error)

func (c *Client) Put(ctx context.Context, key string, body io.Reader, contentType string) error
func (c *Client) Get(ctx context.Context, key string, rangeHeader string) (*s3.GetObjectOutput, error)
func (c *Client) Head(ctx context.Context, key string) (*s3.HeadObjectOutput, error)
```

### Config

| Поле             | Тип    | Описание                                                    |
| ---------------- | ------ | ----------------------------------------------------------- |
| `Endpoint`       | string | URL хранилища. Пусто для AWS S3.                            |
| `Region`         | string | Регион. Например `"ru-central1"`, `"us-east-1"`.            |
| `AccessKey`      | string | Access key для аутентификации.                              |
| `SecretKey`      | string | Secret key для аутентификации.                              |
| `Bucket`         | string | Имя бакета.                                                 |
| `ForcePathStyle` | bool   | Path-style URLs. Требуется для MinIO и большинства non-AWS. |

## Контакты

- Email: morewiktor@yandex.ru
- Telegram: [@MoreWiktor](https://t.me/MoreWiktor)
- GitHub: [@MoreWiktor](https://github.com/MoreWiktor)
