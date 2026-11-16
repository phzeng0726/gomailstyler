# dependency-baseline Specification

## Purpose
定義 gomailstyler 的 Go 版本與相依套件基準，以及升級相依後必須通過的驗證。
## Requirements
### Requirement: Go 版本基準
`go.mod` 的 `go` 指令 SHALL 為 `1.26`（或其 patch 版本），README 的需求章節 MUST 標示 Go 1.26 以上。

#### Scenario: 以 Go 1.26 建置
- **WHEN** 在 Go 1.26 環境執行 `go build ./...`
- **THEN** 建置成功

### Requirement: 相依套件為最新版
所有直接與間接相依 SHALL 升級至執行升級當下的最新發佈版本，`go.sum` 與 `go.mod` 經 `go mod tidy` 整理。

#### Scenario: 檢查可升級套件
- **WHEN** 升級完成後執行 `go list -m -u all`
- **THEN** 本模組 `go.mod` 中列出的套件沒有可用的較新版本

### Requirement: 升級後的驗證
升級 SHALL 在以下檢查全部通過後才算完成：`go build ./...`、`go vet ./...`、`go test ./...`，以及所有 `examples/` 程式可編譯。

#### Scenario: 升級導致行為改變
- **WHEN** 升級後既有單元測試失敗（例如 go-premailer 輸出格式變動）
- **THEN** 必須確認差異來源並記錄於 design.md 或 commit 訊息，只有在確認為上游合理變更時才調整測試預期值

### Requirement: 測試相依僅限測試使用
testify SHALL 只被 `*_test.go` 匯入，production 程式碼不得依賴它。

#### Scenario: 檢查 production 相依
- **WHEN** 執行 `go list -deps .` 與 `go list -deps ./internal/service`（不含測試）
- **THEN** 結果不包含 `github.com/stretchr/testify`

