# kcli Go API

Importable API for programmatic command execution.

```go
import "github.com/kubilitics/kcli/pkg/api"

client := api.NewKCLI(api.Config{})
out, err := client.Execute("get pods -A")
stream, err := client.ExecuteStream("logs app=api --tail=100")
for chunk := range stream {
    if chunk.Done {
        break
    }
    fmt.Println(chunk.Stream, chunk.Data)
}
```
