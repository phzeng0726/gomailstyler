## Context

- gomailstyler 是一個小型 Go 函式庫：`manager.go`（設定、MIME 組裝、`net/smtp` 寄送、Render 委派）與 `internal/service`（`TemplatesService`、`CSSToolsService`）。
- 目前零測試；`go.mod` 為 `go 1.24.2`，相依落後多個 minor 版本（go-premailer v1.24.0 → v1.38.0 跨度最大）。
- 使用者皆為已知的內部使用者，已確認全部升級至 Go 1.26，可直接提高最低 Go 版本。
- `examples/` 與函式庫同屬一個 module，`godotenv` 因此出現在函式庫的 `go.mod`。
- 參考規範來自 peacock-auth-backend（後端分層架構，使用 testify suite + gomock）。

## Goals / Non-Goals

**Goals:**
- 產出適用本專案的 `TESTING_GUIDELINES.md`
- 以「特性測試（characterization test）」方式為現有行為建立安全網，覆蓋率各 package ≥ 85%
- 升級 Go 與所有相依至最新，並用測試偵測升級造成的行為變化

**Non-Goals:**
- 不修正測試中發現的既有缺陷（Header CR/LF 注入、Subject 未做 RFC 2047 編碼、Header 順序隨機、缺 `From`/`Date`、`append(mm.To, mm.Cc...)` 共用底層陣列、HTML part 無 Content-Transfer-Encoding）——留給 `feat/reply` 或後續變更
- 不實作 reply threading Headers 功能
- 不把 `examples/` 拆成獨立 module
- 不導入 CI 設定

## Decisions

### D1. 先寫測試、再升級套件
在舊版相依上先完成測試並全數通過，再執行升級。升級後若有測試失敗，即代表上游行為改變，可明確定位（特別是 go-premailer 的 inline 輸出格式）。
- 替代方案：先升級再寫測試——較省事，但無法分辨行為差異來自升級或本來就如此。
- 實作調整：`go` 指令在寫測試前就先升到 `1.26.0`，因為測試使用 Go 1.26 才有的 `errors.AsType`，在 `go 1.24.2` 下 `go vet` 會報 stdversion 警告。相依套件仍在測試完成後才升級，本決策的目的不受影響。

### D2. 測試框架：testify `suite` + white-box 測試
與使用者其他專案（peacock-auth-backend）一致，使用 `suite.Suite`，測試檔與被測檔同 package，以便直接測 `buildHTMLMessage`、`insertCSSIntoHead` 等未匯出函式及 Manager 內部欄位。
- 替代方案：純標準庫 table-driven——零相依，但與使用者其他專案風格不一致，斷言也較冗長。testify 只出現在測試程式，不影響函式庫使用者的編譯產物。

### D3. 不使用 mock，Service 用真實實作 + `testdata/` fixture
Service 都是純函式性質（讀檔 → 渲染），用真實檔案最能反映行為。fixture 放 `internal/service/testdata/`；錯誤情境（檔案不存在）使用 `t.TempDir()` 或不存在的檔名。Manager 的 Render 委派也透過真實 service + fixture 驗證，不引入 gomock。
- 替代方案：為 `Templates`/`CSSTools` 介面產生 gomock——可隔離 Manager，但對一個只做委派的薄層而言成本大於價值。

### D4. SendMail 以 in-process 假 SMTP server 測試
在測試中以 `net.Listen("tcp", "127.0.0.1:0")` 啟動一個極簡 SMTP server（約百行），支援 `EHLO/HELO`、`AUTH PLAIN`（可設定是否宣告）、`MAIL`、`RCPT`、`DATA`、`QUIT`，並記錄收到的指令與資料。可設定在 RCPT 回 5xx 以測試失敗路徑。
- 不宣告 `STARTTLS`，避免 `smtp.SendMail` 嘗試 TLS。
- `smtp.PlainAuth` 在非 TLS 時只允許 `localhost`/`127.0.0.1`，因此 Manager 的 host 設為 `127.0.0.1` 即可測 AUTH。
- 替代方案：引入第三方 SMTP mock 套件——多一個相依，且控制粒度不如自寫。

### D5. MIME 與 HTML 以解析後比對
- MIME：`net/mail.ReadMessage` 取 header（避開 map 隨機順序），`mime.ParseMediaType` 取 boundary，`multipart.NewReader` 逐 part 比對 header 與內容；base64 分行另以原始 bytes 驗證每行長度。
- HTML（CSS inline 結果）：goquery 解析後比對目標元素的 `style`、`class` 屬性，不比對完整 HTML 字串，以降低對 premailer 序列化細節的耦合。

### D6. 錯誤驗證策略
本專案沒有 sentinel error，規範採三層：
1. 被 `%w` 包裝的標準庫錯誤 → `errors.Is`（如 `fs.ErrNotExist`）
2. 有型別的錯誤 → `errors.AsType`（如 `template.ExecError`、`*textproto.Error`）
3. `errors.New`/字串型錯誤（設定驗證、template 語法錯誤） → 完整錯誤字串精確比對

不為了測試而新增 sentinel error，避免改變 production code。

### D7. 抑制 service 的 log 輸出
Service 以 `log.Printf` 記錄耗時，測試時在 `SetupSuite` 以 `log.SetOutput(io.Discard)` 關閉、`TearDownSuite` 還原。

### D8. 升級方式
1. `go mod edit -go=1.26`（移除 `toolchain` 行，若有）——實際上已在寫測試前完成，見 D1
2. `go get -u -t ./...` 升級直接與間接相依（`-t` 一併升級 testify 等測試相依）
3. `go mod tidy`
4. `go build ./... && go vet ./... && go test ./...`，逐一編譯 `examples/` 下各 `main.go`

## Risks / Trade-offs

- [go-premailer v1.24 → v1.38 跨度大，inline 輸出可能改變] → D1 先寫測試；D5 只比對關鍵屬性；若失敗則比對差異並在 commit 訊息說明
- [特性測試會把既有缺陷「固定」成預期行為] → 這類測試以 `// KNOWN ISSUE:` 註解標示，修正時同步更新測試
- [假 SMTP server 與真實 server 行為不同] → 只驗證 gomailstyler 負責的部分（envelope、AUTH、DATA）；真實寄送仍以 `examples/` 手動驗證
- [部分錯誤分支無法觸發（goquery 解析、premailer 建立/轉換、`doc.Html()`）] → 依規範在測試檔註解列出，覆蓋率門檻設 85% 而非 100%
- [`-race` 在 Windows 需要 cgo] → 規範中列為建議而非必要
- [go-premailer v1.38 新增 `inbucket/html2text` 相依，連帶把 tablewriter、go-json、fatih/color、xxhash 等編進函式庫，使用者的相依數變多] → 目前接受（功能與輸出不受影響，測試已驗證）；若有疑慮可將 go-premailer 固定在 v1.24.x

## Migration Plan

- 函式庫使用者需 Go 1.26 以上（已確認）；公開 API 無變更
- 合併後建議發佈 v1.1.0
- 升級前後版本：

  | 套件 | 升級前 | 升級後 |
  |---|---|---|
  | go（`go.mod` 指令） | 1.24.2 | 1.26.0 |
  | github.com/PuerkitoBio/goquery | v1.10.3 | v1.14.0 |
  | github.com/vanng822/go-premailer | v1.24.0 | v1.38.0 |
  | golang.org/x/text | v0.24.0 | v0.44.0 |
  | golang.org/x/net | v0.39.0 | v0.62.0 |
  | golang.org/x/sys | v0.32.0 | v0.50.0 |
  | github.com/andybalholm/cascadia | v1.3.3 | v1.3.5 |
  | github.com/joho/godotenv | v1.5.1 | v1.5.1（已是最新） |
  | github.com/stretchr/testify | —（新增） | v1.12.1（僅測試） |
- 回滾：還原 `go.mod`/`go.sum` 即可，測試與文件不影響執行時行為
