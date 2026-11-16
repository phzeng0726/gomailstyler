# unit-test-coverage Specification

## Purpose
定義 gomailstyler 公開 API 與內部 service 必須具備的單元測試情境與覆蓋率門檻。
## Requirements
### Requirement: NewManager 設定驗證測試
測試 SHALL 覆蓋 `NewManager` 對 `smtpHost`、`smtpPort`、`smtpSender` 三個必填欄位的驗證，以及 Options 是否正確套用到 Manager。

#### Scenario: 缺少任一必填欄位
- **WHEN** 分別以空 host、空 port、空 sender 呼叫 `NewManager`
- **THEN** 回傳 nil Manager，錯誤字串分別精確等於 `invalid manager config: empty smtpHost`、`invalid manager config: empty smtpPort`、`invalid manager config: empty smtpSender`

#### Scenario: 完整設定
- **WHEN** 以 `WithSMTP`、`WithSMTPPassword`、`WithTemplatePath`、`WithCSSPath` 呼叫 `NewManager`
- **THEN** Manager 的 host/port/sender/password 欄位逐一等於輸入值，且 template 與 CSS 路徑生效（以實際渲染 fixture 驗證）

#### Scenario: 未提供密碼
- **WHEN** 只用 `WithSMTP` 建立 Manager
- **THEN** `smtpPassword` 為 nil

### Requirement: buildHTMLMessage MIME 結構測試
測試 SHALL 解析 `buildHTMLMessage` 的輸出，逐一驗證 header、HTML part、附件 part、inline 圖片 part。

#### Scenario: 最小訊息
- **WHEN** MailMessage 只有 Subject、Message、單一 To
- **THEN** header 依序精確為 `From`、`To`、`Subject`、`Date`、`MIME-Version`、`Content-Type`，其中 `From` 等於 smtpSender、`Date` 等於測試注入的時間、`MIME-Version` 為 `1.0`、`Content-Type` 為 `multipart/mixed` 且 boundary 可解析；不存在 `Cc`、`Message-ID`、`In-Reply-To`、`References` header；multipart 只有一個 part，其 header 為 `Content-Type: text/html; charset="UTF-8"` 與 `Content-Transfer-Encoding: quoted-printable`，解碼後內容等於 Message

#### Scenario: 多位收件者與 Cc
- **WHEN** To 與 Cc 各有兩個地址
- **THEN** `To` 與 `Cc` header 為以逗號連接的精確字串

#### Scenario: 附件與 inline 圖片
- **WHEN** MailMessage 帶一個附件與一個 inline 圖片
- **THEN** part 順序為 HTML、附件、inline 圖片；附件 part 的 `Content-Disposition` 為 `attachment; filename="<name>"` 且無 `Content-ID`；inline part 的 `Content-Disposition` 為 `inline`、`Content-ID` 為 `<cid>`；兩者 base64 解碼後與原始 bytes 相同

#### Scenario: base64 每行 76 字元
- **WHEN** 附件資料編碼後超過 76 字元
- **THEN** 原始 part 內容除最後一行外每行長度精確為 76，最後一行長度為剩餘字元數

#### Scenario: 記錄既有缺陷
- **WHEN** Subject 含 CR/LF 或非 ASCII 字元（原 `KNOWN ISSUE` 情境，已於 add-reply-threading-headers 修正）
- **THEN** 測試作為回歸測試驗證修正後的行為：含 CR/LF 時回傳 `ErrInvalidHeader` 且不產生訊息；非 ASCII 時以 RFC 2047 編碼、解碼後等於原主旨；測試中不再有 `KNOWN ISSUE` 註解

#### Scenario: 完整 threading 訊息
- **WHEN** MailMessage 設定 MessageID、InReplyTo、References、Cc 與兩個自訂 Headers
- **THEN** 整段 header 區塊（固定 boundary 與 Date 後）精確等於預期字串，順序符合 mail-headers 規定

### Requirement: SendMail 寄送測試
測試 SHALL 使用 in-process 假 SMTP server 驗證 `SendMail` 的 SMTP 互動。

#### Scenario: 無密碼寄送
- **WHEN** Manager 未設定密碼並寄出含 To 與 Cc 的訊息
- **THEN** 假 server 收到的 MAIL FROM 等於 smtpSender、RCPT 依序為 To 再 Cc、未收到 AUTH 指令、DATA 等於 `buildHTMLMessage` 的輸出結構

#### Scenario: 空字串密碼
- **WHEN** Manager 以 `WithSMTPPassword("")` 建立
- **THEN** 不進行 AUTH

#### Scenario: 有密碼寄送
- **WHEN** Manager 設定密碼並寄信
- **THEN** 假 server 收到 `AUTH PLAIN`，解碼後帳號為 smtpSender、密碼為設定值

#### Scenario: SMTP 失敗
- **WHEN** 假 server 對 RCPT 回應 5xx，或連線位址無人監聽
- **THEN** `SendMail` 回傳錯誤；5xx 情境以 `errors.AsType[*textproto.Error]` 驗證回應碼

#### Scenario: 不修改呼叫端 To slice
- **WHEN** 呼叫端的 `To` 有剩餘容量且 `Cc` 非空
- **THEN** `SendMail` 後呼叫端底層陣列未被改寫

#### Scenario: 驗證失敗不連線
- **WHEN** MailMessage 驗證失敗
- **THEN** `SendMail` 回傳對應 sentinel error，假 server 的連線計數為 0

### Requirement: TemplatesService 渲染測試
測試 SHALL 覆蓋 `RenderTemplate`、`RenderTemplateWithFuncs`、`RenderHTMLContent`、`RenderHTMLContentWithFuncs` 的成功與錯誤路徑。

#### Scenario: 成功渲染
- **WHEN** 以 fixture template 與資料渲染
- **THEN** 輸出精確等於預期字串

#### Scenario: 預設函式
- **WHEN** template 使用 add、toUpper、toLower、trim、title、formatDate、isEmpty、safeHTML、inc
- **THEN** 每個函式的輸出精確等於預期值（含 safeHTML 不被跳脫、未使用 safeHTML 時被跳脫的對照）

#### Scenario: 自訂函式覆寫預設與只取第一個 FuncMap
- **WHEN** 傳入兩個 FuncMap，第一個覆寫 `toUpper` 並新增函式，第二個定義其他函式
- **THEN** 覆寫與新增生效；第二個 FuncMap 的函式未註冊，使用時回傳解析錯誤

#### Scenario: 錯誤路徑
- **WHEN** template 檔案不存在、template 語法錯誤、執行時錯誤
- **THEN** 檔案不存在以 `errors.Is(err, fs.ErrNotExist)` 驗證；執行錯誤以 `errors.AsType[template.ExecError]` 或等價型別驗證；語法錯誤以精確錯誤字串驗證

### Requirement: CSSToolsService 測試
測試 SHALL 覆蓋 `RenderTemplateWithCSS`、`RenderTemplateWithFuncsAndCSS` 以及 `insertCSSIntoHead`、`convertCSSIntoInline`。

#### Scenario: CSS 轉為 inline style
- **WHEN** 以含 class 的 fixture template 與 CSS 渲染
- **THEN** 以 goquery 解析輸出，目標元素的 `style` 屬性精確等於預期值、`class` 屬性已移除、`!important` 被保留

#### Scenario: template 沒有 head
- **WHEN** template 沒有 `<head>`
- **THEN** 輸出中存在 `<head>` 且 CSS 已被套用

#### Scenario: 錯誤路徑
- **WHEN** CSS 檔不存在或 template 檔不存在
- **THEN** 錯誤以 `errors.Is(err, fs.ErrNotExist)` 驗證，且錯誤字串前綴分別為 `failed to read CSS file <name>:` 與 `failed to render template:`

### Requirement: Manager 渲染方法委派
測試 SHALL 驗證 Manager 的六個 Render 方法實際委派到對應 service（以 fixture 渲染結果驗證），且 `customFuncs` 參數正確傳遞。

#### Scenario: 委派與自訂函式傳遞
- **WHEN** 透過 Manager 呼叫 `RenderTemplateWithFuncs` 並傳入自訂函式
- **THEN** 輸出包含自訂函式的精確結果

### Requirement: 覆蓋率門檻
`go test -cover ./...` SHALL 讓根 package 與 `internal/service` 的 statement 覆蓋率各達 85% 以上。

#### Scenario: 檢查覆蓋率
- **WHEN** 執行 `go test -cover ./...`
- **THEN** 兩個 package 的覆蓋率皆 ≥ 85%

