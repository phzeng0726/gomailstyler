## 1. 測試規範

- [x] 1.1 撰寫根目錄 `TESTING_GUIDELINES.md`（繁中）：三大原則（附本專案範例）、各元件斷言深度表、必要測試分類、無法觸發錯誤路徑的處理、檔案配置/命名/執行指令
- [x] 1.2 在 `CLAUDE.md` 加入測試指令（`go test ./...`、`go test -cover ./...`）與 `TESTING_GUIDELINES.md` 連結

## 2. 測試基礎建設（舊版相依上進行）

- [x] 2.1 `go get github.com/stretchr/testify` 加入測試相依
- [x] 2.2 建立 `internal/service/testdata/`：基本 template、使用全部預設函式的 template、自訂函式 template、語法錯誤 template、執行錯誤 template、含 class 的 CSS template、無 `<head>` 的 template、CSS 檔（含 `!important`）
- [x] 2.3 實作測試用假 SMTP server（`smtp_server_test.go`）：可設定是否宣告 AUTH、RCPT 是否回 5xx，記錄 MAIL FROM、RCPT、AUTH、DATA

## 3. internal/service 測試

- [x] 3.1 `template_test.go`：`RenderTemplate`、`RenderHTMLContent` 成功/檔案不存在/語法錯誤/執行錯誤
- [x] 3.2 `template_test.go`：九個預設函式逐一驗證輸出（含 safeHTML 與跳脫對照）
- [x] 3.3 `template_test.go`：`RenderTemplateWithFuncs`、`RenderHTMLContentWithFuncs` 自訂函式覆寫、新增、只取第一個 FuncMap、無自訂函式
- [x] 3.4 `css_tool_test.go`：`RenderTemplateWithCSS`、`RenderTemplateWithFuncsAndCSS` 成功（goquery 驗 style、class 移除、`!important` 保留）、無 `<head>`、CSS 不存在、template 不存在
- [x] 3.5 `css_tool_test.go`：以註解列出無法觸發的錯誤分支（goquery 解析、`doc.Html()`、premailer 建立/轉換）

## 4. Manager 測試

- [x] 4.1 `manager_test.go`：`NewManager` 三個必填欄位錯誤（精確字串）、完整 Options、未提供密碼
- [x] 4.2 `manager_test.go`：`buildHTMLMessage` 最小訊息、To/Cc 多位、無 Cc 時沒有 Cc header、附件與 inline 圖片的 part 順序與 header、base64 解碼還原、每行 76 字元
- [x] 4.3 `manager_test.go`：`KNOWN ISSUE` 特性測試——Subject 含 CR/LF 未過濾、非 ASCII Subject 未編碼
- [x] 4.4 `manager_test.go`：`SendMail` 無密碼、空字串密碼、有密碼（驗 AUTH PLAIN 帳密）、RCPT 5xx（`errors.AsType[*textproto.Error]` 驗回應碼）、連線失敗
- [x] 4.5 `manager_test.go`：六個 Render 方法的委派與 customFuncs 傳遞
- [x] 4.6 執行 `go test -cover ./...`，確認兩個 package 皆 ≥ 85%，補齊缺漏

## 5. 升級 Go 與相依套件

- [x] 5.1 記錄升級前版本（`go list -m all` 輸出存入 commit 訊息或 PR 描述）
- [x] 5.2 `go mod edit -go=1.26`，`go get -u ./...`，`go mod tidy`
- [x] 5.3 `go build ./...`、`go vet ./...`、`go test ./...` 全數通過；若測試失敗，分析是否為上游行為變更並記錄後再調整
- [x] 5.4 逐一編譯 `examples/` 下所有 `main.go`（`go build ./examples/...`）
- [x] 5.5 確認 `go list -m -u all` 中 `go.mod` 所列套件無可升級版本；確認 production 相依不含 testify

## 6. 文件收尾

- [x] 6.1 README 補上 Go 1.26 以上的需求與執行測試方式
- [x] 6.2 更新 `CLAUDE.md` 的 Critical Implementation Details，加註測試中標記的已知缺陷清單
