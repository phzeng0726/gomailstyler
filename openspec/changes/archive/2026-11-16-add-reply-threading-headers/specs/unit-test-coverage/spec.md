## MODIFIED Requirements

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
