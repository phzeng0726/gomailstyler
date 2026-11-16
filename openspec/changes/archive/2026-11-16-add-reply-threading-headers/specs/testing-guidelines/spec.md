## MODIFIED Requirements

### Requirement: 規範定義三大原則並對應到函式庫情境
規範 SHALL 包含「反向驗證」、「精確斷言」、「錯誤身份驗證」三大原則，且每條原則 MUST 附上以本專案程式碼為例的說明（而非後端 Repository/Service/Handler 的例子）。

#### Scenario: 反向驗證原則以本專案條件為例
- **WHEN** 閱讀反向驗證章節
- **THEN** 規範以 `buildHTMLMessage` 的 `len(mm.Cc) > 0`、`writeHTMLAttachment` 的 `contentID != nil` 等實際條件說明「移除條件時必須有測試失敗」

#### Scenario: 精確斷言原則規定 MIME 與 HTML 的比對方式
- **WHEN** 閱讀精確斷言章節
- **THEN** 規範要求 MIME 訊息以 `net/mail` + `mime/multipart` 解析後逐欄位比對，HTML 輸出以 goquery 解析後比對特定元素的屬性值，不得以 `Contains` 作為唯一斷言；header 順序固定後，也允許以精確字串比對整段 header 區塊（boundary 與 Date 以測試注入的值固定）

#### Scenario: 錯誤身份驗證原則對應本專案的錯誤型態
- **WHEN** 閱讀錯誤身份驗證章節
- **THEN** 規範說明：本專案匯出的 sentinel error（`ErrInvalidHeader`、`ErrReservedHeader`、`ErrInvalidMessageID`）以 `errors.Is` 驗證並搭配完整錯誤字串比對；被 `%w` 包裝的底層錯誤以 `errors.Is`（如 `fs.ErrNotExist`）驗證；有型別的錯誤以 `errors.AsType`（如 `*template.ExecError`）驗證；其餘以 `errors.New` 產生、無 sentinel 的錯誤以完整錯誤字串精確比對；不得僅用 `s.Error(err)`
