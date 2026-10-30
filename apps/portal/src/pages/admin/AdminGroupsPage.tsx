import React, { useCallback, useEffect, useState } from 'react';
import { ApiError, apiClient, createUuidV4, type MemberGroup } from '../../api';
import { ApiErrorAlert } from '../../components/common/ApiErrorAlert';
import { useI18n } from '../../i18n';

export function AdminGroupsPage(): React.ReactElement {
  const { t } = useI18n();
  const [groups, setGroups] = useState<MemberGroup[]>([]);
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const [cursorHistory, setCursorHistory] = useState<string[]>([]);
  const [currentCursor, setCurrentCursor] = useState<string | undefined>(undefined);
  const [error, setError] = useState<unknown>(null);

  const [code, setCode] = useState('');
  const [name, setName] = useState('');
  const [createActionId, setCreateActionId] = useState(
    () => `create-admin-group:${createUuidV4()}`,
  );
  const [conflictServerMsg, setConflictServerMsg] = useState<string | null>(null);
  const [hasConflictError, setHasConflictError] = useState(false);
  const [codeErrDetail, setCodeErrDetail] = useState<{
    field: string;
    reason: string;
  } | null>(null);
  const [nameErrDetail, setNameErrDetail] = useState<{
    field: string;
    reason: string;
  } | null>(null);
  const [validationFields, setValidationFields] = useState<
    Array<{ field: string; reason: string }>
  >([]);

  const loadGroups = useCallback(async (cursor?: string) => {
    try {
      const res = await apiClient.listAdminGroups({ cursor, limit: 50 });
      setGroups(res.items);
      setNextCursor(res.next_cursor ?? null);
    } catch (err) {
      setError(err);
    }
  }, []);

  useEffect(() => {
    void loadGroups(currentCursor);
  }, [loadGroups, currentCursor]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setHasConflictError(false);
    setConflictServerMsg(null);
    setCodeErrDetail(null);
    setNameErrDetail(null);
    setValidationFields([]);

    try {
      await apiClient.createAdminGroup({ code, name }, createActionId);
      setCode('');
      setName('');
      setCreateActionId(`create-admin-group:${createUuidV4()}`);
      await loadGroups(currentCursor);
    } catch (err) {
      setError(err);
      if (err instanceof ApiError) {
        if (err.status === 409 || err.code === 'STATE_CONFLICT') {
          setHasConflictError(true);
          setConflictServerMsg(err.serverMessage ?? null);
        } else if (err.status === 422 || err.code === 'VALIDATION_FAILED') {
          let fields = err.fieldErrors;
          if (fields.length === 0) {
            const inferred: Array<{ field: string; reason: string }> = [];
            if (!/^[a-z0-9][a-z0-9_-]{0,63}$/.test(code)) {
              inferred.push({ field: '/code', reason: 'pattern' });
            }
            if (!name.trim() || name.length > 100) {
              inferred.push({ field: '/name', reason: 'invalid' });
            }
            if (inferred.length === 0) {
              inferred.push({
                field: '/body',
                reason: err.serverMessage ?? 'validation_failed',
              });
            }
            fields = inferred;
          }
          setValidationFields(fields);
          const codeErr = fields.find((f) => f.field === '/code' || f.field === 'code');
          const nameErr = fields.find((f) => f.field === '/name' || f.field === 'name');
          if (codeErr) {
            setCodeErrDetail(codeErr);
          }
          if (nameErr) {
            setNameErrDetail(nameErr);
          }
        }
      }
      setCreateActionId(`create-admin-group:${createUuidV4()}`);
    }
  };

  return (
    <div data-testid="admin-groups-page">
      <div className="page-header">
        <div>
          <h1 className="page-title">{t('admin.groups.title')}</h1>
          <p className="page-subtitle">{t('admin.groups.subtitle')}</p>
        </div>
      </div>

      <ApiErrorAlert error={error} testId="groups-error" />

      <div className="card">
        <h2 className="section-title">{t('admin.groups.createTitle')}</h2>
        <form onSubmit={(e) => void handleCreate(e)} noValidate>
          {hasConflictError && (
            <div
              className="alert alert-danger"
              role="alert"
              data-testid="group-conflict-field-error"
              style={{ marginBottom: '0.75rem' }}
            >
              {conflictServerMsg
                ? `${t('admin.groups.conflictError')} (${conflictServerMsg})`
                : t('admin.groups.conflictError')}
            </div>
          )}
          {validationFields.length > 0 && (
            <div
              className="alert alert-warning"
              role="alert"
              data-testid="group-validation-detail"
              style={{ marginBottom: '0.75rem' }}
            >
              <ul style={{ margin: 0, paddingLeft: '1.25rem' }}>
                {validationFields.map((fe, idx) => (
                  <li key={`${fe.field}-${idx}`}>
                    <code>{fe.field}</code>: {fe.reason}
                  </li>
                ))}
              </ul>
            </div>
          )}

          <div
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fit, minmax(260px, 1fr))',
              gap: '1rem',
            }}
          >
            <div className="form-group">
              <label className="form-label" htmlFor="group-code-input">
                {t('admin.groups.codeLabel')}
              </label>
              <input
                id="group-code-input"
                data-testid="group-code-input"
                type="text"
                value={code}
                onChange={(e) => {
                  setCode(e.target.value);
                  setHasConflictError(false);
                  setConflictServerMsg(null);
                  setCodeErrDetail(null);
                  setCreateActionId(`create-admin-group:${createUuidV4()}`);
                }}
                placeholder="asia-premium"
              />
              {codeErrDetail && (
                <div
                  className="text-danger"
                  style={{ fontSize: '0.82rem', marginTop: '4px' }}
                  data-testid="group-code-field-error"
                >
                  {`${t('admin.groups.validationCodeError')} [${codeErrDetail.field}: ${codeErrDetail.reason}]`}
                </div>
              )}
            </div>

            <div className="form-group">
              <label className="form-label" htmlFor="group-name-input">
                {t('admin.groups.nameLabel')}
              </label>
              <input
                id="group-name-input"
                data-testid="group-name-input"
                type="text"
                value={name}
                onChange={(e) => {
                  setName(e.target.value);
                  setHasConflictError(false);
                  setConflictServerMsg(null);
                  setNameErrDetail(null);
                  setCreateActionId(`create-admin-group:${createUuidV4()}`);
                }}
                placeholder="Asia Premium Exits"
              />
              {nameErrDetail && (
                <div
                  className="text-danger"
                  style={{ fontSize: '0.82rem', marginTop: '4px' }}
                  data-testid="group-name-field-error"
                >
                  {`${t('admin.groups.validationNameError')} [${nameErrDetail.field}: ${nameErrDetail.reason}]`}
                </div>
              )}
            </div>
          </div>

          <button
            type="submit"
            className="btn btn-primary"
            data-testid="group-create-submit-btn"
          >
            {t('admin.groups.createBtn')}
          </button>
        </form>
      </div>

      <div className="card">
        <div className="table-wrap">
          <table data-testid="groups-table">
            <thead>
              <tr>
                <th>{t('admin.groups.col.code')}</th>
                <th>{t('admin.groups.col.name')}</th>
                <th>{t('admin.groups.col.groupId')}</th>
                <th>{t('admin.groups.col.exitCount')}</th>
              </tr>
            </thead>
            <tbody>
              {groups.length === 0 ? (
                <tr>
                  <td colSpan={4} className="text-muted">
                    {t('admin.groups.empty')}
                  </td>
                </tr>
              ) : (
                groups.map((g) => (
                  <tr
                    key={g.group_id}
                    data-testid={`group-row-${g.code}`}
                    data-group-id={g.group_id}
                  >
                    <td className="mono" style={{ fontWeight: 700 }}>
                      {g.code}
                    </td>
                    <td>{g.name}</td>
                    <td className="mono" style={{ fontSize: '0.82rem' }}>
                      {g.group_id}
                    </td>
                    <td>
                      <span
                        className="badge badge-info mono"
                        data-testid={`group-exit-count-${g.code}`}
                      >
                        {g.exit_count}
                      </span>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {(cursorHistory.length > 0 || nextCursor) && (
          <div
            style={{
              display: 'flex',
              justifyContent: 'flex-end',
              gap: '0.5rem',
              marginTop: '1rem',
            }}
          >
            <button
              type="button"
              className="btn btn-secondary"
              data-testid="groups-prev-page-btn"
              disabled={cursorHistory.length === 0}
              onClick={() => {
                const prev = cursorHistory[cursorHistory.length - 1];
                setCursorHistory((h) => h.slice(0, -1));
                setCurrentCursor(prev || undefined);
              }}
            >
              {t('common.pagination.prev')}
            </button>
            <button
              type="button"
              className="btn btn-secondary"
              data-testid="groups-next-page-btn"
              disabled={!nextCursor}
              onClick={() => {
                if (nextCursor) {
                  setCursorHistory((h) => [...h, currentCursor ?? '']);
                  setCurrentCursor(nextCursor);
                }
              }}
            >
              {t('common.pagination.next')}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
