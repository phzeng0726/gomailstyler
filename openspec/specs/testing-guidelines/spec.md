# testing-guidelines Specification

## Purpose
定義本專案單元測試的撰寫規範（TESTING_GUIDELINES.md）必須涵蓋的原則、斷言深度、測試分類與執行方式。
## Requirements
### Requirement: 測試規範文件存在於專案根目錄
專案 SHALL 在根目錄提供 `TESTING_GUIDELINES.md`，以繁體中文撰寫，作為本專案所有單元測試的撰寫依據。

#### Scenario: 開發者查閱測試規範
- **WHEN** 開發者要為 gomailstyler 新增或修改測試
- **THEN** 可在專案根目錄找到 `TESTING_GUIDELINES.md`，且 `CLAUDE.md` 有連結指向它

### Requirement: 規範定義三大原則並對應到函式庫情境
規範 SHALL 包含「反向驗證」、「精確斷言」、「錯誤身份驗證」三大原則，且每條原則 MUST 附上以本專案程式碼為例的說明（而非後端 Repository/Service/Handler 的例子）。

#### Scenario: 反向驗證原則以本專案條件為例
- **WHEN** 閱讀反向驗證章節
- **THEN** 規範以 `buildHTMLMessage` 的 `len(mm.Cc) > 0`、`writeHTMLAttachment` 的 `contentID != nil` 等實際條件說明「移除條件時必須有測試失敗」

#### Scenario: 精確斷言原則規定 MIME 與 HTML 的比對方式
- **WHEN** 閱讀精確斷言章節
- **THEN** 規範要求 MIME 訊息以 `net/mail` + `mime/multipart` 解析後逐欄位比對，HTML 輸出以 goquery 解析後比對特定元素的屬性值，不得以 `Contains` 作為唯一斷言

#### Scenario: 錯誤身份驗證原則對應本專案的錯誤型態
- **WHEN** 閱讀錯誤身份驗證章節
- **THEN** 規範說明：被 `%w` 包裝的底層錯誤以 `errors.Is`（如 `fs.ErrNotExist`）驗證；有型別的錯誤以 `errors.AsType`（如 `*template.ExecError`）驗證；本專案以 `errors.New` 產生、無 sentinel 的錯誤以完整錯誤字串精確比對；不得僅用 `s.Error(err)`

### Requirement: 規範定義各元件的斷言深度
規範 SHALL 以表格列出 Manager（設定/Options）、MIME 組裝、SMTP 寄送、Template 渲染、CSS inline 各元件的最低斷言要求。

#### Scenario: SMTP 寄送的斷言深度
- **WHEN** 閱讀 SMTP 寄送列
- **THEN** 規範要求以測試內建的假 SMTP server 驗證 envelope sender、RCPT 收件者清單（含 Cc）、是否進行 AUTH 及其帳密、DATA 內容

### Requirement: 規範定義必要測試分類
規範 SHALL 列出每個公開方法必須涵蓋的測試分類：最小 happy path、完整 happy path、邊界/分支、輸入錯誤、上游依賴失敗（檔案不存在、template 解析/執行失敗、SMTP 失敗）。

#### Scenario: 無法觸發的錯誤路徑
- **WHEN** 某個錯誤分支在正常輸入下無法觸發（如 goquery 解析失敗）
- **THEN** 規範要求在測試檔中以註解列出該分支及無法觸發的原因，而非為了測試修改 production code

### Requirement: 規範定義檔案配置、命名與執行方式
規範 SHALL 規定測試檔與被測檔同目錄同 package（white-box）、fixture 置於 `testdata/`、測試框架使用 testify `suite`、測試命名格式，以及執行測試與覆蓋率的指令。

#### Scenario: 執行全部測試
- **WHEN** 開發者依規範執行 `go test ./...`
- **THEN** 所有測試不需要網路、不需要 `.env`、不需要真實 SMTP server 即可通過

