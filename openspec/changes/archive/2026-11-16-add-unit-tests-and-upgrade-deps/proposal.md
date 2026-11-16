## Why

gomailstyler 目前沒有任何單元測試（零個 `*_test.go`），任何修改（例如接下來 `feat/reply` 要加的郵件 Header 功能）都無法驗證是否破壞既有行為。同時 `go.mod` 停在 `go 1.24.2`，相依套件也落後多個版本（goquery v1.10.3 → v1.14.0、go-premailer v1.24.0 → v1.38.0、x/net v0.39.0 → v0.62.0 等），累積了安全性修正與相容性落差。先建立測試安全網並升級版本，再進行功能開發。

## What Changes

- 新增 `TESTING_GUIDELINES.md`：以 peacock-auth-backend 的三大原則（反向驗證、精確斷言、錯誤身份驗證）為基礎，改寫成適用於「Go 函式庫」的版本，涵蓋 MIME 訊息組裝、SMTP 寄送、template 渲染、CSS inline 四個面向的測試深度要求
- 為現有程式碼補上單元測試：
  - 根目錄 `manager_test.go`：`NewManager` 設定驗證、Options 套用、`buildHTMLMessage` MIME 結構、`writeHTMLAttachment` base64 分行、Render 方法委派
  - `internal/service/template_test.go`：四個 Render 方法、預設 template 函式、自訂函式覆寫
  - `internal/service/css_tool_test.go`：CSS 插入 `<head>`、premailer inline 轉換、各錯誤路徑
- 引入測試相依 `github.com/stretchr/testify`（僅測試使用）
- `go` 指令版本升級至 `1.26`；所有直接與間接相依升級至最新版
- 測試只「記錄現有行為」，不修改 production code 的行為；測試過程中發現的既有缺陷（如 Header CR/LF 注入、Header 順序隨機、`append` 共用底層陣列）以測試註解標記並列入後續工作，不在本變更修正
- **BREAKING**（對下游使用者）：`go` 最低版本由 1.24 提高為 1.26，使用者需 Go 1.26 以上才能編譯

## Capabilities

### New Capabilities

- `testing-guidelines`：本專案單元測試撰寫規範（原則、各元件斷言深度、必要測試分類、命名與檔案配置）
- `unit-test-coverage`：現有公開 API 與內部 service 必須具備的測試案例與覆蓋範圍
- `dependency-baseline`：Go 版本與相依套件版本基準，以及升級後必須通過的驗證（build、vet、test、範例編譯）

### Modified Capabilities

（無，`openspec/specs/` 目前沒有既有 spec）

## Impact

- **新增檔案**：`TESTING_GUIDELINES.md`、`manager_test.go`、`internal/service/template_test.go`、`internal/service/css_tool_test.go`、測試用 fixture（`internal/service/testdata/`）
- **修改檔案**：`go.mod`、`go.sum`、`CLAUDE.md`（補上測試指令）、`README.md`（Go 版本需求）
- **相依套件**：goquery、go-premailer、x/text、x/net、cascadia 等升級；新增 testify（test-only）
- **公開 API**：無變更
- **發版**：建議以 v1.1.0 發佈（最低 Go 版本提高）
