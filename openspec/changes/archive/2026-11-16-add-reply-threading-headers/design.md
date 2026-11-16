## Context

- `buildHTMLMessage` 目前把 header 放進 `map[string]string` 後 range 輸出（順序隨機），值不做任何檢查或編碼，回傳 `[]byte` 沒有 error。
- 上一個變更已建立完整測試（根 package 100% 覆蓋）與 6 個 `KNOWN ISSUE` 特性測試，可直接作為這次修正的回歸基準。
- 下游 peacock-mail-backend 的需求：第一封自訂 `Message-ID`（如 `<maintenance.{noticeID}.{userID}@domain>`），第二封帶 `In-Reply-To` / `References` 與 `RE: ` 主旨，Outlook/Exchange 與 Gmail 都要歸串。
- 已決定：專用欄位 + Headers map；保留名稱衝突回傳 error；未給 MessageID 時不輸出；全部 6 個 KNOWN ISSUE 一併修正。

## Goals / Non-Goals

**Goals:**
- 讓呼叫端能設定 threading header 與任意自訂 header
- 所有 header 輸出都經過驗證（無 CR/LF、名稱合法、非保留）與正確編碼
- 原始訊息可預測：固定順序、可注入時間，測試能精確比對整段 header
- 修正全部 KNOWN ISSUE，並把對應測試改寫為正確行為

**Non-Goals:**
- 不自動產生 Message-ID，也不改變 `SendMail` 的回傳簽名
- 不支援 `Thread-Index`（Outlook 專有的二進位格式）；需要時可透過 Headers 自行設定
- 不處理 To/Cc 的顯示名稱編碼（`"名字" <addr>`），維持原樣輸出（僅檢查 CR/LF）
- 不新增 Bcc 支援
- 不修改 template / CSS 相關功能

## Decisions

### D1. API：專用欄位 + Headers map
```go
type MailMessage struct {
    // ...existing fields...
    MessageID  string            // with or without angle brackets
    InReplyTo  string            // Message-ID of the parent mail
    References []string          // thread ancestry, oldest first
    Headers    map[string]string // extra headers; reserved names are rejected
}
```
專用欄位讓函式庫統一處理角括號、格式驗證、References 串接與折行，呼叫端不會在這些細節出錯。Headers map 保留彈性（`X-*`、`Thread-Topic`）。
- 替代方案：只有 Headers map——每個呼叫端都要自己處理 `<id>` 與 References 格式，容易做錯。

### D2. Sentinel error 放在新檔 `errors.go`
```go
var (
    ErrInvalidHeader    = errors.New("invalid header")
    ErrReservedHeader   = errors.New("reserved header")
    ErrInvalidMessageID = errors.New("invalid message id")
)
```
以 `fmt.Errorf("%w: %s %q", ErrXxx, field, value)` 包裝，訊息格式固定，例如 `invalid message id: MessageID "no-at-sign"`、`reserved header: "subject"`、`invalid header: Subject contains CR or LF`。
- 替代方案：自訂 `*HeaderError` 型別帶欄位——資訊更多，但 sentinel 已足夠讓呼叫端分辨類別，且與 Go 標準庫風格一致。

### D3. header 組裝改用有序 slice，抽到 `header.go`
- 新增 `type headerField struct{ name, value string }`，`(m *Manager) buildHeaders(mm, boundary string) ([]headerField, error)` 依規定順序產生並完成所有驗證與編碼（boundary 由 `buildHTMLMessage` 傳入，用於 `Content-Type`）；`buildHTMLMessage` 只負責寫出 header 與 body。
- `buildHeaders` 再拆出兩個純函式：`buildThreadingHeaders(mm)` 處理 `Message-ID` / `In-Reply-To` / `References`，`buildCustomHeaders(headers)` 處理自訂 header 的驗證、排序與編碼，方便個別測試。
- 驗證函式：`validateHeaderValue(field, value)`（CR/LF）、`validateHeaderName(name)`、`normalizeMessageID(field, id) (string, error)`、`isReservedHeader(name)`。
- 自訂 header 依 `strings.ToLower(name)` 排序；同時用小寫名稱偵測重複。
- 替代方案：繼續用 map 並在輸出前排序——內建 header 的順序規則仍需另外處理，不如直接用有序 slice 清楚。

### D4. `buildHTMLMessage` 簽名改為 `([]byte, error)`，時間可注入
- 未匯出函式，改簽名不影響公開 API。`SendMail` 先呼叫它，失敗就直接回傳，不連線 SMTP。
- `Manager` 新增未匯出欄位 `now func() time.Time`，`NewManager` 設為 `time.Now`；測試（white-box）覆寫為固定時間，Date 可精確比對。
- 替代方案：新增公開 Option `WithClock`——對使用者沒有價值，只為測試而擴大 API。

### D5. 編碼
- Subject 與自訂 header 值：`mime.QEncoding.Encode("UTF-8", v)`；純 ASCII 時標準庫原樣回傳，符合「ASCII 原樣輸出」。
- References 折行：逐個 ID 累加，若加入下一個 ID 會讓目前實體行超過 78 字元，就插入 `\r\n `。`net/mail` 解析時會把折行還原成空白。
- HTML part：`Content-Transfer-Encoding: quoted-printable`，用 `quotedprintable.NewWriter` 寫入 buffer 並 `Close()`。

### D6. multipart boundary 測試方式
`multipart.NewWriter` 的 boundary 是隨機的。為了讓「完整 threading 訊息」情境能精確比對整段 header，測試只比對 `Content-Type` 以外的 header 區塊並另外驗證 boundary 格式，或在 white-box 測試中以正規表示式把 boundary 替換為固定字串後再比對。不為了測試改成可注入 boundary。

### D7. `SendMail` 收件者清單
```go
allRecipients := make([]string, 0, len(mm.To)+len(mm.Cc))
allRecipients = append(allRecipients, mm.To...)
allRecipients = append(allRecipients, mm.Cc...)
```

### D8. 假 SMTP server 新增連線計數
`fakeSMTPServer` 增加 `connections int`（在 Accept 時遞增，透過 `Record()` 讀取），用來驗證「驗證失敗不連線」。

## Risks / Trade-offs

- [原始訊息格式改變，下游若有比對原始 MIME 的測試會失敗] → 屬於修正，README 與 release notes 說明；公開 API 只有新增
- [先前可寄出的 CR/LF 輸入現在回傳錯誤] → 這類輸入本來就會產生錯誤郵件，回傳錯誤才正確；release notes 標示
- [`mime.QEncoding` 對很長的中文主旨會產生多個 encoded-word 但不折行，單行可能超過 78 字元] → 仍遠低於 998 字元硬上限，可接受；未來需要時再加折行
- [Outlook/Exchange 的 conversation view 主要依主旨與 Thread-Index，不保證只靠 References 就歸串] → README 說明需保持主旨一致（`RE: ` + 原主旨），必要時可透過 Headers 設定 `Thread-Topic`
- [From 只輸出 smtpSender，無顯示名稱] → 與 envelope sender 一致、最不容易被判為偽冒；顯示名稱留待後續需求

## Migration Plan

- 函式庫使用者：升級後不需改程式即可繼續使用；若要 threading，設定新欄位
- peacock-mail-backend：
  1. 第一封：`MessageID: fmt.Sprintf("maintenance.%d.%d@<domain>", noticeID, userID)`
  2. 第二封：`InReplyTo` 設為同一 ID、`Subject: "RE: " + 原主旨`
- 發佈下一個 minor 版本；回滾即退回前一版
