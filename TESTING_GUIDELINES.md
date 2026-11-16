# Unit Test 撰寫規範

本文件是 gomailstyler 所有單元測試的撰寫依據。gomailstyler 是 Go 函式庫，沒有 Repository / Service / Handler 分層，因此以「設定、MIME 組裝、SMTP 寄送、Template 渲染、CSS inline」五個元件來定義測試要求。

## 三大原則

### 1. 反向驗證 (Reverse Validation)

每個 production code 的 `if` 條件，至少要有一個測試會在該條件被移除或反轉時失敗。

| Production 條件 | 必須存在的測試 |
|---|---|
| `manager.go` `validate()` 的三個 `== ""` | 每個欄位為空各一個測試，驗證錯誤字串 |
| `buildHTMLMessage` 的 `len(mm.Cc) > 0` | 有 Cc → 有 `Cc` header；無 Cc → **沒有** `Cc` header |
| `writeHTMLAttachment` 的 `contentID != nil` | 附件 → 無 `Content-ID`；inline 圖片 → `Content-ID: <cid>` |
| base64 分行的 `end > len(encoded)` | 資料長度剛好 76 的倍數、與非倍數各一個測試 |
| `SendMail` 的 `smtpPassword != nil && *smtpPassword != ""` | 未設定密碼、空字串密碼、有密碼三種情況 |
| `insertCSSIntoHead` 的 `head.Length() == 0` | template 有 `<head>` 與沒有 `<head>` 各一個測試 |
| service 的 `len(customFuncs) > 0` | 不傳自訂函式、傳一個、傳多個（只有第一個生效） |

邊界值測試：剛好在限制上、超過限制各一個，確保測到的是 `>` 而不是 `>=`（或反之）。

### 2. 精確斷言 (Precise Assertion)

不得只用 `NoError` / `NotEmpty` / `Contains` 作為唯一斷言，每個測試都要驗證具體的輸出值。

- **字串結果**（template 渲染）：用 `s.Equal(expected, got)` 精確比對完整字串
- **MIME 訊息**（`buildHTMLMessage` 輸出）：
  - 以 `net/mail.ReadMessage` 解析 header 後逐欄位 `Equal`。**不得**比對整段 header 文字，因為目前 header 是從 map 輸出，順序隨機
  - 以 `mime.ParseMediaType` 取出 boundary，再用 `multipart.NewReader` 逐一讀取 part，驗證 part 數量、順序、每個 part 的 header 與內容
  - 附件與 inline 圖片：base64 解碼後與原始 bytes 比對
- **HTML 結果**（CSS inline）：以 goquery 解析後，對目標元素的 `style`、`class` 屬性做精確比對；不比對整份 HTML，避免和 premailer 的序列化細節綁死
- **集合結果**（RCPT 收件者清單等）：驗證長度與每個元素的值和順序

### 3. 錯誤身份驗證 (Error Identity Verification)

所有 error path 測試都必須驗證錯誤的具體身份，不得只寫 `s.Error(err)`。本專案沒有自訂 sentinel error，依錯誤來源分三種方式：

| 錯誤來源 | 驗證方式 | 範例 |
|---|---|---|
| 被 `%w` 包裝的標準庫錯誤 | `errors.Is` | 檔案不存在：`s.ErrorIs(err, fs.ErrNotExist)` |
| 有型別的錯誤 | `errors.AsType` 並驗證欄位 | `execErr, ok := errors.AsType[template.ExecError](err)`；SMTP 回應碼：`errors.AsType[*textproto.Error](err)` 後驗證 `.Code` |
| `errors.New` / 字串型錯誤 | 完整字串比對 `s.EqualError` | `s.EqualError(err, "invalid manager config: empty smtpHost")` |

被包裝的錯誤也要驗證外層訊息前綴，例如 `failed to read CSS file style.css:`，確保錯誤是從預期的分支回傳。

## 各元件 Assertion 深度要求

| 元件 | 要求 |
|---|---|
| **Manager 設定 / Options** | 每個必填欄位缺漏各一個測試，用 `EqualError` 驗證；Options 套用後逐欄位驗證 Manager 內部欄位（white-box）；`smtpPassword` 要區分 nil 與空字串 |
| **MIME 組裝** | 依「精確斷言」的方式解析；驗證 `MIME-Version`、`To`、`Subject`、`Content-Type`、`Cc` 每個 header；驗證 part 順序固定為 HTML → 附件 → inline 圖片；驗證每個 part 的 `Content-Type`、`Content-Transfer-Encoding`、`Content-Disposition`、`Content-ID`；base64 每行 76 字元 |
| **SMTP 寄送** | 使用測試內建的假 SMTP server（`smtp_server_test.go`），驗證 `MAIL FROM`、`RCPT` 清單與順序（To 在前、Cc 在後）、是否送出 `AUTH` 及解碼後的帳密、`DATA` 內容可被解析成正確的 MIME 訊息；失敗路徑驗證 SMTP 回應碼 |
| **Template 渲染** | 輸出用完整字串比對；九個預設函式各自驗證輸出值；自訂函式的覆寫與新增都要驗證；檔案不存在、語法錯誤、執行錯誤三種錯誤各一個測試 |
| **CSS inline** | goquery 驗證目標元素的 `style` 精確值、`class` 已移除、`!important` 有保留；CSS 檔與 template 檔不存在各一個測試，驗證外層訊息前綴與 `fs.ErrNotExist` |

## 每個公開方法的必要測試分類

| 分類 | 說明 |
|---|---|
| Happy path（最小） | 只帶必填欄位，例如只有 Subject / Message / 一個 To 的 MailMessage |
| Happy path（完整） | 帶齊所有欄位（Cc、附件、inline 圖片、密碼、自訂函式），逐欄位驗證 |
| 分支 / 邊界值 | 對應「反向驗證」表格中的每個條件 |
| 輸入錯誤 | 設定缺漏、template 語法錯誤 |
| 上游依賴失敗 | 檔案不存在、template 執行失敗、SMTP 回應 5xx、SMTP 連線失敗 |

### 無法觸發的錯誤分支

有些錯誤分支在正常輸入下無法觸發，例如 goquery 解析失敗、`doc.Html()` 失敗、premailer 建立或轉換失敗。遇到這種情況：

- **不要**為了測試去修改 production code（例如為了注入錯誤而加介面）
- 在測試檔開頭以註解列出這些分支和無法觸發的原因：

```go
// Untestable error branches:
// - insertCSSIntoHead: goquery.NewDocumentFromReader never fails on a strings.Reader
// - convertCSSIntoInline: premailer.Transform errors only on internal parser failures
```

### 已知缺陷的特性測試

測試記錄的是「目前的行為」。如果目前行為本身是缺陷（例如 Subject 中的 CR/LF 沒有被過濾），仍然寫測試記錄實際輸出，並加上 `// KNOWN ISSUE:` 註解說明正確行為應該是什麼。修正缺陷時要同步更新這些測試。

## 檔案配置與命名

- 測試檔與被測檔放在同一目錄、同一 package（white-box），才能直接測試 `buildHTMLMessage`、`insertCSSIntoHead` 等未匯出函式

  ```
  gomailstyler/
  ├── manager.go
  ├── manager_test.go
  ├── smtp_server_test.go          # 假 SMTP server（只在測試中編譯）
  └── internal/service/
      ├── template.go
      ├── template_test.go
      ├── css_tool.go
      ├── css_tool_test.go
      └── testdata/                # template、CSS fixture
  ```

- 測試框架：testify `suite`，每個被測型別一個 suite，例如 `ManagerSuite`、`TemplatesServiceSuite`、`CSSToolsServiceSuite`
- 測試方法命名：`Test<方法>_<情境>`，例如 `TestBuildHTMLMessage_WithoutCc`、`TestSendMail_RcptRejected`
- 同一方法的多個相似情境用 table-driven 搭配 `s.Run(name, func() {...})`
- fixture 放在 `testdata/`；需要「檔案不存在」時使用 `s.T().TempDir()` 或不存在的檔名，不要刪除 fixture
- service 會用 `log.Printf` 輸出耗時，在 `SetupSuite` 中以 `log.SetOutput(io.Discard)` 關閉，`TearDownSuite` 還原
- testify 只能在 `*_test.go` 中匯入，production 程式碼不得依賴

## 執行方式

```bash
# 執行全部測試（不需要網路、.env 或真實 SMTP server）
go test ./...

# 覆蓋率（根 package 與 internal/service 各需 ≥ 85%）
go test -cover ./...

# 產生覆蓋率報告
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out

# 建議：race 檢查（Windows 需要 cgo / gcc）
go test -race ./...
```
