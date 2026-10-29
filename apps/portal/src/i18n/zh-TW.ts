/**
 * Traditional Chinese (zh-TW) locale dictionary — primary UI language.
 * Preserves exact wording from `docs/spec/PROJECT_SPEC.zh-TW.md` (§7.3, §9.2, §11.6, §12.6, §13.12, §18).
 */

export const zhTW = {
  // App Shell & Navigation
  'app.title': 'Freedom VPN 控制台與會員中心',
  'app.skipToMain': '跳至主要內容',
  'app.language.label': '介面語言',
  'app.language.zhTW': '繁體中文 (zh-TW)',
  'app.language.en': 'English (en)',
  'app.role.admin': '管理員 (Admin)',
  'app.role.member': '會員 (Member)',
  'app.mockSwitcher.label': '模擬身分切換 (TEST ONLY)：',
  'app.mockSwitcher.anonymous': '未登入訪客 (Anonymous)',
  'app.mockSwitcher.adminNoMfa': '管理員：未完成 MFA (Admin without MFA)',
  'app.mockSwitcher.admin': '管理員 (Admin)',
  'app.mockSwitcher.memberActive': '會員：有效 (Active / 正常速度)',
  'app.mockSwitcher.memberThrottled': '會員：有效 (FUP 調速中 / 含缺口)',
  'app.mockSwitcher.memberPending': '會員：待付款確認 (pending_payment)',
  'app.mockSwitcher.memberCancelEnd': '會員：期末取消 (cancel_at_period_end)',
  'app.mockSwitcher.memberGrace': '會員：寬限期 (past_due / grace)',
  'app.mockSwitcher.memberExpired': '會員：已到期 (expired)',
  'app.mockSwitcher.memberSuspended': '會員：人工停權 (suspended)',
  'app.mockSwitcher.trigger401': '模擬登入階段過期 (401)',
  'app.logout': '登出',
  'app.footer.securityNote':
    '本前端不保存任何 NetBird 管理 Token、發行簽章私鑰或支付卡號資料（§18.4, AGENTS.md §2）。',

  // Navigation items
  'nav.admin.heading': '管理後台 (§18.1)',
  'nav.admin.overview': '總覽',
  'nav.admin.nodes': '節點列表',
  'nav.admin.addNode': '新增節點啟用碼',
  'nav.admin.updates': '發行與全部更新',
  'nav.admin.members': '會員列表與管理',
  'nav.admin.fup': '方案與公平使用 (FUP)',
  'nav.admin.dns': '網站與 DNS 政策',
  'nav.admin.audit': '操作與稽核紀錄',
  'nav.member.heading': '會員中心 (§18.4)',
  'nav.member.service': '我的服務',
  'nav.member.gettingStarted': '開始使用',
  'nav.member.exits': '可用出口',
  'nav.member.devices': '我的裝置',
  'nav.member.usage': '用量與速度',
  'nav.member.billing': '付款與訂閱',
  'nav.member.dnsProtection': '網站防護說明',

  // Auth & Guards
  'auth.anonymous.title': '尚未登入 (Unauthenticated)',
  'auth.anonymous.description':
    '請透過單一登入 (OIDC) 驗證身分後再進入控制台或會員中心。',
  'auth.anonymous.loginBtn': '前往 OIDC 單一登入',
  'auth.mfaRequired.title': '需要完成管理員多因素驗證 (MFA Required)',
  'auth.mfaRequired.description':
    '您的帳號具備管理員角色，但尚未完成本階段的多因素驗證 (mfa_verified: false)。依安全契約（§16.1），所有管理後台操作均強制要求 MFA。',
  'auth.mfaRequired.verifyBtn': '前往完成 MFA 驗證',
  'auth.sessionExpired.title': '登入階段已過期 (401)',
  'auth.sessionExpired.description':
    '您的 OIDC 登入階段已失效或過期，請重新驗證身分後再繼續操作。',
  'auth.sessionExpired.reloginBtn': '重新登入恢復階段',
  'auth.forbidden.title': '存取遭拒：權限不足 (403)',
  'auth.forbidden.description':
    '您目前的帳號角色為會員 (Member)，無權進入管理員專用路徑 (/admin/*)。此前端路由守衛僅為縱深防禦，所有後端 /api/v1/admin/* 端點均強制驗證 Admin 角色與 MFA（§9.1, §16.1）。',
  'auth.forbidden.backToMember': '返回會員中心 (/member)',
  'errorBoundary.title': '畫面發生非預期錯誤',
  'errorBoundary.reset': '重新載入頁面狀態',

  // Common Operation & Conflict UI (§14.8, §18.3)
  'op.savedPendingApply': '已儲存，等待套用',
  'op.acceptedPending': '已受理，等待背景與節點確認（202 Accepted 不代表已完成）',
  'op.completedApplied': '已套用（已取得實際觀測確認）',
  'op.partiallySucceeded': '部分完成（請檢視下方步驟與租約限制說明）',
  'op.failed': '套用失敗',
  'op.superseded': '已被較新操作取代 (Superseded)',
  'op.routeWithdrawnDeviceUnconfirmed': '中央已撤銷路由；設備撤權尚未確認／受租約限制',
  'api.error.title': 'API 請求錯誤',
  'api.error.codeLabel': '錯誤代碼 (code)：{code}',
  'api.error.requestIdLabel': '請求追蹤 ID (request_id)：{requestId}',
  'api.error.retryAfterNotice':
    '請求頻率或相依服務受限，請於 {seconds} 秒後重試 (Retry-After: {seconds}s)',
  'common.nextPage': '載入下一頁 (Next Cursor)',
  'common.resetPage': '回到第一頁',
  'op.idLabel': '操作 ID (operation_id)',
  'op.requestIdLabel': '請求追蹤 ID (request_id)',
  'op.generationsLabel':
    'Desired Generation: {desired} ／ Observed Generation: {observed}',
  'op.step.agent_policy': '節點政策下發與 ACK (agent_policy)',
  'op.step.netbird_route_readback': 'NetBird 路由配置與讀回驗證 (netbird_route_readback)',
  'op.step.data_path_probe': '合成資料路徑與隔離探測 (data_path_probe)',
  'op.step.node_allowset_withdraw': '節點會員轉送集合移除 (node_allowset_withdraw)',
  'op.step.cert_and_peer_revoke': '機器憑證與 NetBird Peer 撤銷 (cert_and_peer_revoke)',
  'op.step.policy_outbox_sync': '會員權益與節點政策同步 (policy_outbox_sync)',
  'op.step.resolver_1_smoke_apply':
    '解析器 #1 語法檢查與原子切換 (resolver_1_smoke_apply)',
  'op.step.resolver_2_smoke_apply':
    '解析器 #2 語法檢查與原子切換 (resolver_2_smoke_apply)',
  'op.dismiss': '關閉進度面板',

  'conflict.dialog.title': '偵測到資源版本衝突 (HTTP 412 Precondition Failed)',
  'conflict.dialog.description':
    '此資源在您編輯期間已被其他管理員或背景同步更新。為防止靜默覆寫他人設定，系統已拒絕本次儲存（§14.5, §18.3）。請重新載入最新狀態後再重新檢視修改。',
  'conflict.dialog.currentVersion':
    '伺服器目前資源版本 (current_resource_version)：{version}',
  'conflict.dialog.reloadBtn': '重新載入最新資源（不覆寫）',

  // Admin Overview (§18.1)
  'admin.overview.title': '系統營運總覽',
  'admin.overview.subtitle':
    '分別呈現出口可用性、控制連線、隔離安全、更新狀態與會員資格；不以單一綠點代表所有功能正常（§16.5, §18.1）。',
  'admin.overview.asOf': '資料最後更新時間 (UTC)：{asOf}',
  'admin.overview.card.availableExits': '可提供服務出口數',
  'admin.overview.card.availableExitsDesc':
    '通過身分、上架意圖、配置、隔離安全與支援版本檢查（§5.1）',
  'admin.overview.card.controlStatus': '節點控制連線觀測',
  'admin.overview.card.controlBreakdown':
    '在線 {online} ／ 延遲 {stale} ／ 離線 {offline}',
  'admin.overview.card.safetyFailed': '隔離或安全基線異常節點',
  'admin.overview.card.safetyFailedDesc': '在線但安全檢查失敗，已自動停止會員轉送',
  'admin.overview.card.enrollmentsPending': '待處理註冊或配置失敗',
  'admin.overview.card.enrollmentsPendingDesc': '尚在註冊或本機配置異常的設備',
  'admin.overview.card.updateAttention': '需留意的節點更新結果',
  'admin.overview.card.updateAttentionDesc':
    '更新失敗、已回復 (rolled_back) 或等待離線補更新',
  'admin.overview.card.members': '有效會員數',
  'admin.overview.card.membersBreakdown':
    '有效 {active}（調速中 {throttled}）／ 停權或到期 {inactive}',
  'admin.overview.link.goNodes': '前往節點列表',
  'admin.overview.link.goUpdates': '前往發行與更新',
  'admin.overview.link.goMembers': '前往會員管理',

  // Admin Nodes List (§5, §18.1)
  'admin.nodes.title': '節點列表 (Nodes)',
  'admin.nodes.subtitle':
    '所有來源（預裝盒子、雲端 VM、既有 Ubuntu 主機）皆使用相同資料模型與管理契約（NODE01, NODE04）。',
  'admin.nodes.addBtn': '新增節點啟用碼',
  'admin.nodes.fleetUpdateBtn': '前往一次更新全部節點',
  'admin.nodes.filter.search': '搜尋名稱或地區',
  'admin.nodes.filter.publication': '發布意圖 (Desired)',
  'admin.nodes.filter.control': '控制連線狀態',
  'admin.nodes.filter.all': '全部',
  'admin.nodes.col.node': '節點名稱 ／ 地區 ／ 架構',
  'admin.nodes.col.publication': '發布意圖 vs 實際觀測',
  'admin.nodes.col.controlOnline': '1. 控制在線',
  'admin.nodes.col.dataAvailable': '2. 資料路徑可用',
  'admin.nodes.col.policyValid': '3. 政策與隔離有效',
  'admin.nodes.col.updateComplete': '4. 更新狀態與版本',
  'admin.nodes.col.summary': '服務狀態摘要 (§18.1)',
  'admin.nodes.col.actions': '操作',
  'admin.nodes.viewDetail': '檢視與管理',

  'node.summary.online_serving': '在線，正常提供出口服務',
  'node.summary.online_safety_failed_not_serving': '在線，隔離檢查失敗，未提供服務',
  'node.summary.online_disabled_not_serving': '在線，管理員已停用出口，未提供服務',
  'node.summary.ready_unpublished_not_serving': '已就緒，尚未上架，未提供服務',
  'node.summary.offline_route_withdrawn_unconfirmed':
    '中央已撤銷路由；設備撤權尚未確認／受租約限制',
  'node.summary.rolled_back_serving': '在線，上次更新已回復至最後良好版本 (last_good)',
  'node.summary.revoked_not_serving': '裝置身分已永久撤銷，停止服務',

  // Admin Node Detail (§18.1, §18.3)
  'admin.nodeDetail.back': '← 返回節點列表',
  'admin.nodeDetail.title': '節點詳細資訊：{name}',
  'admin.nodeDetail.meta':
    'Node ID: {nodeId} ｜ Incarnation: {incarnationId} ｜ ETag: {etag}',
  'admin.nodeDetail.desiredVsObserved.title':
    '期望狀態 (Desired) 與實際觀測狀態 (Observed) 對照',
  'admin.nodeDetail.field.setting': '設定項目',
  'admin.nodeDetail.field.desired': '中央期望狀態 (Desired)',
  'admin.nodeDetail.field.observed': '節點與路由實際觀測 (Observed)',
  'admin.nodeDetail.field.publication': '出口發布狀態',
  'admin.nodeDetail.field.configGen': '設定版本 (config_generation)',
  'admin.nodeDetail.field.release': '程式發行版本 (Release)',
  'admin.nodeDetail.field.memberGroups': '可用會員群組',
  'admin.nodeDetail.field.capacity': '節點總頻寬上限 (上／下行)',
  'admin.nodeDetail.field.safetyAndIp': 'Internet-only 隔離檢查 ／ 觀測出口 IP',
  'admin.nodeDetail.lastAckAt': '最後成功套用時間：{time}',
  'admin.nodeDetail.pubActions.title': '上架、停用與撤銷身分操作 (§5.2, §18.3)',
  'admin.nodeDetail.pubActions.publishBtn': '上架出口 (Publish)',
  'admin.nodeDetail.pubActions.disableBtn': '停用出口轉送 (Disable / Unpublish)',
  'admin.nodeDetail.pubActions.revokeSectionTitle':
    '永久撤銷機器身分 (Revoke Node Identity)',
  'admin.nodeDetail.pubActions.revokeDistinctNote':
    '注意：撤銷機器身分不同於暫停上架。撤銷會立即使 Agent mTLS 憑證、NetBird Peer 與路由全部失效；重新加入必須建立新的單次啟用碼。歷史用量帳本與稽核紀錄仍會保留（§5.2, §18.3）。',
  'admin.nodeDetail.pubActions.revokeBtn': '永久撤銷機器身分',
  'admin.nodeDetail.editForm.title': '修改節點分類與容量設定 (需要 If-Match ETag)',
  'admin.nodeDetail.editForm.name': '顯示名稱',
  'admin.nodeDetail.editForm.region': '地區標示',
  'admin.nodeDetail.editForm.groups': '允許會員群組 (逗號分隔)',
  'admin.nodeDetail.editForm.upBps': '上行容量上限 (bps 十進位字串)',
  'admin.nodeDetail.editForm.downBps': '下行容量上限 (bps 十進位字串)',
  'admin.nodeDetail.editForm.notes': '來源與維運備註 (不影響權限)',
  'admin.nodeDetail.editForm.saveBtn': '儲存節點設定',
  'admin.nodeDetail.editForm.simulateConflictBtn': '模擬他人同時修改衝突 (觸發 412)',
  'admin.nodeDetail.errors.title': '最近錯誤與操作紀錄（點擊展開技術細節）',
  'admin.nodeDetail.errors.empty': '目前無異常紀錄',
  'admin.nodeDetail.history.title': '節點更新歷史 (Update History)',

  // Admin Add Node / Enrollment (§18.2)
  'admin.enrollment.title': '新增節點與單次啟用碼 (§18.2)',
  'admin.enrollment.subtitle':
    '有效啟用碼代表加入已獲授權，設備自動完成配置與隔離檢查後即可由管理員上架，無需第二次人工入網核准（ENR01, ENR04）。',
  'admin.enrollment.form.name': '預填節點名稱（可選）',
  'admin.enrollment.form.region': '預填地區（可選）',
  'admin.enrollment.form.groups': '預填會員群組（逗號分隔）',
  'admin.enrollment.form.expiresHours':
    '啟用碼有效時數（小時，建議預設：預裝出貨 168 小時 / 7 天）',
  'admin.enrollment.form.notes': '備註（例如：預裝盒子／OCI VM，不參與權限判斷）',
  'admin.enrollment.form.submitBtn': '產生單次節點啟用碼',
  'admin.enrollment.oneTime.bannerTitle':
    '單次啟用碼已建立（僅顯示這一次，離開本畫面後無法再次讀取明文）',
  'admin.enrollment.oneTime.securityNote':
    '中央資料庫僅保存雜湊值，且回應標示 Cache-Control: no-store。此啟用碼不會寫入瀏覽器儲存空間、網址參數或日誌。請立即安全複製並寫入單台設備設定，切勿直接貼入會留存於 shell history 的明文命令列（§18.2）。',
  'admin.enrollment.oneTime.copyBtn': '安全複製啟用碼',
  'admin.enrollment.oneTime.copied': '已複製至剪貼簿',
  'admin.enrollment.oneTime.viewStatusBtn':
    '模擬離開並重新進入啟用碼狀態頁（驗證明文不再顯示）',
  'admin.enrollment.detail.title': '啟用碼狀態查詢',
  'admin.enrollment.detail.secretGone':
    '基於安全契約（§14.2, §14.5, §18.2），原始啟用碼明文僅在建立當下顯示一次，目前無法讀回（secret_available: false）。若遺失請撤銷此啟用碼並重新建立。',
  'admin.enrollment.detail.stateLabel': '目前狀態：{state}',
  'admin.enrollment.detail.revokeBtn': '撤銷未使用的啟用碼',

  // Admin Fleet Update (§13.12)
  'admin.updates.title': '發行版本與全部節點更新 (§13.12)',
  'admin.updates.subtitle':
    '選擇一個已核准且簽章完整的 Release，按一次「全部更新」即納入當下所有未撤銷／未退役節點。目標版本與最低允許服務版本 (minimum_serving_version) 分開管理（UPD01, UPD04）。',
  'admin.updates.currentFleetDefault':
    '目前全域預設目標版本 (fleet_default_release_id)：{releaseId}',
  'admin.updates.minServingVersion':
    '最低允許服務版本 (minimum_serving_version)：{version}',
  'admin.updates.selectReleaseLabel': '選擇目標發行版本 (Target Release)',
  'admin.updates.restartRequiredYes':
    '需要重啟資料面服務 (requires_service_restart: true) — 切換期間會短暫暫停轉送',
  'admin.updates.restartRequiredNo':
    '不需重啟資料面服務 (requires_service_restart: false)',
  'admin.updates.runAllBtn': '一次更新全部節點 (scope=all_nodes)',
  'admin.updates.acceptedBanner':
    '已受理：批次工作已建立，正在追蹤各節點進度（HTTP 202 不代表更新已完成）',
  'admin.updates.batchTitle': '更新批次進度：{batchId}',
  'admin.updates.batchStateLabel': '批次整體狀態：{state}',
  'admin.updates.targetCountLabel': '不可變快照目標總數 (target_count)：{count}',
  'admin.updates.bucketSumCheck': '互斥狀態桶加總：{sum} / {targetCount}',
  'admin.updates.verifiedSuccessLabel':
    '已驗證成功數 (succeeded + already_current)：{count}（注意：rolled_back、superseded、cancelled 絕不計入成功，§13.12）',
  'admin.updates.retryFailedBtn': '重試失敗目標 (Retry Failed)',
  'admin.updates.col.node': '節點 ／ 地區 ／ 架構',
  'admin.updates.col.pubIntent': '發布意圖 (更新不改變意圖)',
  'admin.updates.col.versions': '目前版本 → 目標版本 (Last Good)',
  'admin.updates.col.bucket': '互斥進度桶 (Bucket)',
  'admin.updates.col.error': '錯誤碼 ／ 最後回報時間',

  // Admin Members (§9.2, §18.1)
  'admin.members.title': '會員列表與資格管理 (§9.2, §18.1)',
  'admin.members.subtitle':
    '顯示有效服務資格 (Entitlement)、已付期間、人工授權 (Grant)、裝置名額與跨節點月用量十進位彙總。',
  'admin.members.filter.state': '篩選資格狀態',
  'admin.members.col.member': '會員 ／ 信箱',
  'admin.members.col.entitlement': '資格狀態 (Entitlement)',
  'admin.members.col.planAndEnd': '方案 ／ 有效服務截止 (effective_service_end)',
  'admin.members.col.devices': '已授權裝置數',
  'admin.members.col.usage': '本月累計用量 (十進位 GB / GiB)',
  'admin.members.col.fupTier': '公平使用級距 (FUP)',
  'admin.members.col.payment': '付款與最後對帳狀態',
  'admin.members.col.actions': '操作',
  'admin.members.manageBtn': '管理會員',

  'admin.memberDetail.back': '← 返回會員列表',
  'admin.memberDetail.title': '會員詳細管理：{name}',
  'admin.memberDetail.entitlementBox': '資格與期限分解 (§9.2, §15.2)',
  'admin.memberDetail.paidThrough': '帳務已付截止 (paid_through)：{time}',
  'admin.memberDetail.effectiveEnd': '有效服務截止 (effective_service_end)：{time}',
  'admin.memberDetail.reconciledAt': '最後供應商對帳時間：{time}',
  'admin.memberDetail.suspendSection': '人工停權與恢復 (優先於付款成功，§9.2)',
  'admin.memberDetail.suspendReasonLabel': '停權原因說明',
  'admin.memberDetail.suspendBtn': '立即人工停權 (Suspend Member)',
  'admin.memberDetail.resumeBtn': '解除人工停權 (Resume Member)',
  'admin.memberDetail.grantSection':
    '新增有限期人工授權 (Finite Entitlement Grant, §14.2)',
  'admin.memberDetail.grantValidUntil': '授權截止時間 (UTC RFC 3339，必填且禁止無限期)',
  'admin.memberDetail.grantReason': '授權理由（必填，例如：移轉並行期或補償）',
  'admin.memberDetail.grantSubmitBtn': '建立有限期授權',
  'admin.memberDetail.devicesSection': '會員已登錄裝置與撤銷 (§9.4)',
  'admin.memberDetail.revokeDeviceBtn': '撤銷此裝置',

  // Admin FUP & Plans (§11.1, §18.1)
  'admin.fup.title': '方案與公平使用政策草稿編輯器 (§11.1)',
  'admin.fup.testOnlyBanner':
    'TEST ONLY（待決策）：以下所有流量門檻、速率上限與價格參照均為測試佔位值，非正式營運承諾（AGENTS.md §1, §11）。',
  'admin.fup.modeLabel': '政策執行模式 (mode)',
  'admin.fup.mode.shadow': 'Shadow Mode（僅計算預期級距與影響摘要，資料面維持原速）',
  'admin.fup.mode.enforce': 'Enforce Mode（實際下發節點 HTB + fq_codel 雙向限速）',
  'admin.fup.effectiveAtLabel': '生效時間 (effective_at, UTC RFC 3339)',
  'admin.fup.minRateLabel': '最低可用保證速率下限 (minimum_rate_bps，十進位字串)',
  'admin.fup.reasonLabel': '修訂理由',
  'admin.fup.tiersTitle': '級距定義（必須由 0 開始、門檻嚴格遞增、上下行齊備且非負）',
  'admin.fup.addTierBtn': '+ 新增級距',
  'admin.fup.removeTierBtn': '移除',
  'admin.fup.validateAndPublishBtn': '驗證並發布新政策 Revision',
  'admin.fup.validationPassed':
    '草稿驗證通過：門檻嚴格遞增、上下行速率有效且無級距空檔。',
  'fup.validation.emptyTiers': '至少必須定義一個級距（Tier 0）。',
  'fup.validation.missingZeroStart':
    'Tier 0 的起始門檻 (threshold_bytes) 必須為 "0"，不得留有未定義空檔。',
  'fup.validation.invalidThreshold': '門檻必須為合法的非負整數十進位字串。',
  'fup.validation.nonIncreasingThreshold':
    '各級距的 threshold_bytes 必須嚴格遞增（T_i > T_{i-1}）。',
  'fup.validation.missingDirectionRate':
    '每個級距都必須同時填寫下行 (down_bps) 與上行 (up_bps) 速率。',
  'fup.validation.invalidRateDecimal': '上／下行速率必須為合法的非負十進位整數字串。',
  'fup.validation.rateBelowMinimum':
    '上／下行速率不得低於最低可用速率 (minimum_rate_bps)。',
  'fup.validation.invalidMinimumRate': '最低可用速率必須為大於 0 的十進位整數字串。',
  'fup.validation.invalidPriority': '排程優先序 (priority) 必須為 1 至 10 的整數。',
  'fup.validation.emptyTierName': '每個級距都必須提供中英文顯示名稱。',
  'fup.validation.invalidEffectiveAt': '生效時間必須為 UTC RFC 3339 格式（以 Z 結尾）。',

  // Admin DNS Policy (§12.4, §12.5)
  'admin.dns.title': '網站與 DNS 防護政策 (§12.4, §12.5)',
  'admin.dns.subtitle':
    '管理網域級強制封鎖、允許清單與威脅情報來源。任何 DNS 允許規則均不得解除固定 Internet-only 內網隔離基線（ISO04, §12.4）。',
  'admin.dns.resolversTitle': '受控 DNS 解析器套用狀態',
  'admin.dns.publishBtn': '編譯並發布政策 Bundle (202)',
  'admin.dns.rollbackBtn': '回復至上一良好版本 (last_good)',
  'admin.dns.addRuleTitle': '新增網域規則（自動 IDNA 正規化、轉小寫、移除尾端 dot）',
  'admin.dns.domainInput': '網域名稱（禁止輸入 http://、路徑 /path、埠號或萬用字元 *）',
  'admin.dns.actionInput': '動作',
  'admin.dns.action.block': '強制封鎖 (Priority 1: Admin Block)',
  'admin.dns.action.allow': '明確允許 (Priority 2: Admin Allow，解除分類名單誤擋)',
  'admin.dns.includeSubdomains':
    '涵蓋所有子網域 (include_subdomains，按完整 label 邊界匹配)',
  'admin.dns.reasonInput': '規則原因',
  'admin.dns.addRuleSubmit': '新增規則至草稿',
  'admin.dns.previewTitle': '規則匹配與優先序預覽工具 (§12.4 Match Preview)',
  'admin.dns.previewInput':
    '輸入測試查詢網域（例如 sub.tracker.example.com 或 nottracker.example.com）',
  'admin.dns.previewOutcome.blocked_admin':
    '結果：阻擋 (NXDOMAIN) — 命中優先序 1：管理員強制封鎖',
  'admin.dns.previewOutcome.allowed_admin':
    '結果：放行解析 — 命中優先序 2：管理員明確允許',
  'admin.dns.previewOutcome.blocked_threat_feed':
    '結果：阻擋 (NXDOMAIN) — 命中優先序 3：安全與惡意網域名單',
  'admin.dns.previewOutcome.resolved_normal':
    '結果：一般解析 — 優先序 4：轉送安全上游解析器',
  'admin.dns.previewIsolationWarning':
    '最高優先序提醒：若解析結果指向 RFC1918、宿主內網或雲端 metadata 位址，仍會被節點 IP 層 Internet-only 防火牆強制阻擋（§8.1, §12.4）。',
  'dns.error.EMPTY_DOMAIN': '請輸入網域名稱。',
  'dns.error.URL_OR_PATH_NOT_ALLOWED':
    '僅接受純網域名稱，不接受 URL scheme (https://)、埠號 (:443)、路徑 (/path) 或查詢參數（§12.4）。',
  'dns.error.WILDCARD_NOT_ALLOWED':
    '不接受萬用字元 (*)，請使用 include_subdomains 勾選框控制子網域匹配（§12.4）。',
  'dns.error.INVALID_LABEL':
    '網域 label 格式無效，必須為包含至少兩個有效 DNS label 的名稱。',

  // Admin Audit (§14.2, §18.1)
  'admin.audit.title': '操作與安全稽核紀錄 (§14.2, §18.1)',
  'admin.audit.subtitle':
    '唯讀、已去敏的設定與權限變更歷史（append-only）。依規格不提供任何刪除歷史紀錄的按鈕（§18.1）。',
  'admin.audit.filter.resource': '資源類型篩選',
  'admin.audit.filter.search': '搜尋 Request ID / 操作者 / 動作',
  'admin.audit.col.time': '發生時間 (UTC)',
  'admin.audit.col.actor': '操作者 ／ 角色',
  'admin.audit.col.action': '動作 ／ 資源',
  'admin.audit.col.trace': 'Request ID ／ Operation ID',
  'admin.audit.col.result': '結果與去敏前後差異',

  // Member Portal (§18.4)
  'member.service.title': '我的服務 (§18.4)',
  'member.service.stateLabel': '目前服務資格狀態',
  'member.service.reasonLabel': '狀態說明與原因',
  'member.service.planLabel': '目前方案',
  'member.service.effectiveEndLabel': '有效服務到期日 (effective_service_end)',
  'member.service.paidThroughLabel': '已付費期間截止 (paid_through)',
  'member.service.nextResetLabel': '下一次公平使用 (FUP) 重置時間',
  'member.service.fupThrottledBanner':
    '本期大量使用，速度已依公平使用政策調整；下次重置為…… {resetAt}',

  'member.gettingStarted.title': '開始使用與平台安裝指引 (§7.3, §18.4)',
  'member.gettingStarted.subtitle':
    '本服務使用官方 NetBird 應用程式搭配自架管理端點與單一登入 (OIDC)，首次使用無需人工傳送 WireGuard 私鑰。',
  'member.gettingStarted.mgmtUrlLabel': '自架管理伺服器網址 (Self-hosted Management URL)',
  'member.gettingStarted.copyUrlBtn': '複製管理網址',
  'member.gettingStarted.osSelectLabel': '選擇您的作業系統平台',
  'member.gettingStarted.step1': '1. 下載並安裝該平台的官方 NetBird 客戶端程式。',
  'member.gettingStarted.step2':
    '2. 打開應用程式設定，將 Management URL 改為上方自架網址。',
  'member.gettingStarted.step3':
    '3. 點選 SSO 登入，使用與本入口網站相同的帳號完成驗證與裝置授權。',
  'member.gettingStarted.step4':
    '4. 在應用程式的出口選單中，手動選擇您要使用的地區出口節點。',
  'member.gettingStarted.killSwitchWarningTitle':
    '重要限制說明：無完整 Always-on ／ Kill-Switch 保證 (§7.3)',
  'member.gettingStarted.killSwitchWarningBody':
    '官方 NetBird 客戶端目前不提供完整的系統級 always-on／kill-switch 模式。本平台的伺服器端隔離僅保護經過 VPN 隧道的流量；若 VPN 隧道斷線或切換網路，您的裝置可能會直接使用原本的本地 ISP 或 Wi-Fi 網路連線。',
  'member.gettingStarted.androidPrivateDnsWarning':
    'Android 特別注意事項（§12.6）：Android 系統的「私人 DNS (Private DNS)」功能會繞過 VPN 的受控 DNS 設定。請於 Android「設定 → 網路和網際網路 → 私人 DNS」將其設為「關閉」，以確保網站惡意網域防護正常生效。',

  'member.exits.title': '可用出口節點 (§18.4, §18.5)',
  'member.exits.manualSelectionNotice':
    '重要提示：出口切換必須在您裝置上的 NetBird 應用程式內手動選擇。本網站僅顯示您的方案目前可用的出口清單與健康狀態，不會也無法從網頁遠端替您的應用程式切換出口；清單順序亦不代表自動最快節點（EXIT02, §18.4）。',
  'member.exits.col.name': '出口名稱',
  'member.exits.col.region': '地區',
  'member.exits.col.status': '目前可用狀態',
  'member.exits.col.notes': '連線與使用說明',

  'member.devices.title': '我的裝置 (§9.4, §18.4)',
  'member.devices.quotaSummary': '已授權裝置名額使用量：{used} / {limit} 台',
  'member.devices.quotaHelp':
    '若已達方案裝置上限，您可在此撤銷不再使用或遺失的舊裝置，待同步完成後即可登錄新裝置（§7.3, AT02）。您只能管理及撤銷自己的裝置。',
  'member.devices.col.name': '裝置名稱 ／ 平台',
  'member.devices.col.overlayIp': '分配位址 ／ Peer ID',
  'member.devices.col.status': '授權狀態',
  'member.devices.col.lastSeen': '最近活動 ／ 授權時間',
  'member.devices.col.action': '操作',
  'member.devices.revokeBtn': '撤銷此裝置',

  'member.usage.title': '用量與速度 (§10.1, §11.6, §18.4)',
  'member.usage.honestyBanner':
    '誠實計量說明：此畫面顯示「已記錄的 VPN 用量 (forward_admission_v1)」，並非即時跳動的流量儀表。資料隨節點批次上報與中央彙總更新（§10.1, §18.4）。',
  'member.usage.asOfLabel': '彙總資料時間 (aggregate_as_of)：{asOf}',
  'member.usage.periodLabel': '計量期間 (UTC 自然月)：{period}',
  'member.usage.totalLabel': '本期累計總用量 (上行 + 下行)',
  'member.usage.uploadLabel': '本期上行用量 (upload_bytes)',
  'member.usage.downloadLabel': '本期下行用量 (download_bytes)',
  'member.usage.speedStatusLabel': '目前速度狀態與公平使用級距',
  'member.usage.rateCapsLabel': '目前同一出口合計速率上限：下行 {down} ／ 上行 {up}',
  'member.usage.nextResetLabel': '下次重置時間：{resetAt}',
  'member.usage.gapWarningTitle': '計量資料品質提醒 ({quality})',
  'member.usage.gapWarningBody':
    '本期間記錄含計量缺口或跨界估算：{note}。系統僅呈現已驗證事件，不會以虛構數字填補缺口（§10.7, §18.4）。',

  'member.billing.title': '付款與訂閱 (§9.3, §18.4, AT03)',
  'member.billing.noCardNotice':
    '安全承諾：本平台絕不在自家網頁表單收集信用卡號或付款秘密。所有付款與訂閱管理均透過支付供應商的受託管安全頁面 (Hosted Checkout / Portal) 進行（§18.4）。',
  'member.billing.checkoutBtn': '前往支付供應商託管頁面付款 (Hosted Checkout)',
  'member.billing.portalBtn': '開啟支付供應商帳單管理入口 (Billing Portal)',
  'member.billing.simulateReturnBtn': '模擬從支付商回跳頁返回 (?checkout=return, AT03)',
  'member.billing.returnPendingBadge': '確認中',
  'member.billing.returnPendingBody':
    '您已從支付供應商頁面返回，目前狀態為「確認中」。依據安全契約（BILL03, AT03, §18.4），瀏覽器回跳網址本身不作為開通依據；系統正在等待支付供應商簽章 Webhook 與後端對帳完成後，才會更新您的正式使用資格。',

  'member.dns.title': '網站防護說明 (§12.1, §12.6, §12.8, §18.4)',
  'member.dns.scopeTitle': '第一版防護範圍：僅限 DNS 網域層級保護',
  'member.dns.scopeBody':
    '本服務透過受控 DNS 解析器比對惡意、釣魚與自訂封鎖網域名單，於名稱解析階段阻擋已知風險網域。本服務不進行 TLS 中間人解密 (MITM)，不會掃描 HTTPS 加密連線內的網頁內容、下載檔案或完整網址路徑 (/path)。',
  'member.dns.limitationsTitle': '終端設備限制：Private DNS 與 DoH (DNS over HTTPS)',
  'member.dns.limitationsBody':
    '若您的手機開啟 Android「私人 DNS (Private DNS)」，或瀏覽器／應用程式自行使用加密 DNS (DoH / DoT) 或直接連線 IP 位址，查詢將不會經過本服務的受控 DNS 解析器，網域防護亦不會生效（§12.6）。此外，規則更新後，作業系統與瀏覽器既有快取或長連線可能需待快取到期或重新啟動應用後才會生效。',
  'member.dns.falsePositiveTitle': '誤擋回報與協助',
  'member.dns.falsePositiveBody':
    '若您發現正常網站遭誤擋，請記下完整網域名稱（不需提供完整瀏覽網址路徑）並聯繫管理團隊（support@example.com）進行允許清單審核。',

  // Wave 3 Live Control Plane Additions
  'member.service.closedBanner':
    '您的帳號與服務資格已結清關閉 (account_status: closed)。所有裝置授權已撤銷，歷史帳務與稽核紀錄依保留政策封存。',
  'member.service.usageUnavailableNotice':
    '目前無法載入最新彙總用量（將不會阻斷您的服務狀態檢視）。',
  'member.billing.planSelectLabel': '選擇訂閱方案 (Plan ID)',
  'member.billing.retryCheckoutBtn': '使用相同冪等鍵重試前往付款 (Retry Checkout)',
  'member.devices.addGuideTitle': '如何新增與授權裝置 (§7.3, §9.4)',
  'member.devices.addGuideBody':
    '請在新裝置安裝官方 NetBird 客戶端並使用本帳號單一登入 (SSO)。控制平面收到登入後會建立待授權操作 (202 Accepted) 並在背景完成群組與對等節點授權。',
  'admin.nodeDetail.pubActions.unpublishBtn': '設為未上架 (Unpublish)',
  'admin.nodeDetail.retireSectionTitle': '退役並移除節點服務 (Retire Node, DELETE)',
  'admin.nodeDetail.retireNote':
    '退役會撤銷憑證、移除路由並將節點標記為 retired；已在進行中的更新批次目標會轉為 cancelled（§13.12）。',
  'admin.nodeDetail.retireReasonLabel': '退役原因說明',
  'admin.nodeDetail.retireBtn': '執行節點退役 (Retire Node)',
  'admin.nodeDetail.editForm.clearCapacity': '清除節點容量上限 (clear_capacity)',
  'admin.enrollment.form.delivery': '交付模式 (delivery)',
  'admin.enrollment.form.deliveryInstaller':
    '一鍵安裝腳本 (installer — 預設 15 分鐘有效)',
  'admin.enrollment.form.deliveryPreinstalled':
    '預裝設備出貨 (preinstalled — 預設 7 天有效)',
  'admin.enrollment.form.reinstallNodeId':
    '重裝既有節點 ID（選填 node_id，保留邏輯節點並換發新 Incarnation，AT07）',
  'admin.updates.registerReleaseTitle':
    '登錄已簽章發行版本 (Register Signed Release, §14.2)',
  'admin.updates.registerReleaseBtn': '登錄新發行版本',
  'admin.updates.needsAttentionBadge': '需人工留意 (needs_attention)',
  'admin.updates.unconfirmedBadge': '逾時未確認 (unconfirmed)',
  'admin.updates.noBatchSelected':
    '目前尚無進行中或已選取的更新批次。請於上方選擇目標版本並按「一次更新全部節點」建立批次。',
  'admin.memberDetail.patchSection':
    '修改會員裝置上限覆寫與備註 (PATCH /admin/members/{id}, If-Match)',
  'admin.memberDetail.deviceLimitOverrideLabel':
    '單一會員裝置上限覆寫 (device_limit_override，1–20)',
  'admin.memberDetail.clearDeviceLimitOverride': '清除裝置上限覆寫（回復方案預設值）',
  'admin.memberDetail.notesLabel': '管理員內部備註 (notes)',
  'admin.memberDetail.savePatchBtn': '儲存會員設定 (If-Match)',
  'admin.fup.overridesTitle':
    '單一會員臨時公平使用例外覆寫 (Temporary FUP Overrides, §11.1)',
  'admin.fup.addOverrideBtn': '+ 新增臨時例外覆寫',
  'admin.fup.planSectionTitle':
    '建立方案政策修訂 (Create Plan Policy Revision, schema: plan)',
  'admin.fup.createPlanBtn': '建立方案 Revision',
  'admin.policies.listTitle': '已登錄政策修訂列表與啟用管理 (Policy Revisions)',
  'admin.policies.activateBtn': '預覽差異並啟用 (Activate with Diff)',
  'admin.policies.confirmDialogTitle':
    '確認啟用政策修訂與變更差異對照 (Confirm Policy Activation & Diff)',
  'admin.policies.confirmActivateBtn': '確認送出啟用 (Send If-Match & Activate)',
  'admin.policies.cancelActivateBtn': '取消',
  'admin.dns.otherSchemasTitle':
    'DNS Profile、威脅情報名單與發行支援政策修訂 (dns_profile / threat_feed / release_support)',
  'common.pagination.prev': '← 上一頁',
  'common.pagination.next': '下一頁 →',
  'member.devices.addGuide':
    '如何新增與授權裝置 (§7.3, §9.4)：請在新裝置安裝官方 NetBird 客戶端並使用本帳號單一登入 (SSO)。控制平面收到登入後會建立待授權操作 (202 Accepted) 並在背景完成群組與對等節點授權。',
  'admin.nodeDetail.pubActions.retireBtn': '執行節點退役 (Retire Node)',
  'admin.memberDetail.deviceLimitLabel': '顯示名稱 (display_name)',
  'admin.memberDetail.clearDeviceLimit': '會員群組 (member_group_ids，逗號分隔)',
  'admin.fup.overrideMemberId': '會員 ID (member_id)',
  'admin.fup.overrideValidUntil': '有效截止時間 (valid_until, UTC)',
  'admin.policies.schemaSelectLabel': '選擇政策 Schema 類型',
  'admin.policies.createRevisionTitle': '建立政策修訂 (Create Policy Revision)',
  'admin.updates.manifestUrlLabel': 'Manifest URL (受信任儲存庫位址)',
  'admin.updates.manifestJsonLabel': 'Signed Manifest Envelope JSON',
} as const;

export type TranslationKey = keyof typeof zhTW;
