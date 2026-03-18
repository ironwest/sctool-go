import { useState } from 'react';
import {
  OpenCSVFileDialog,
  LoadCSVFile,
  LoadProcessedCSV,
  OpenCSVFileDialog as OpenDialog,
} from '../../../wailsjs/go/main/App';
import { useWizardStore } from '../../store/wizardStore';

export function Step1Upload() {
  const { yearLabel, setCsvLoadResult, setStep } = useWizardStore();
  const csvLoadResult = useWizardStore((s) => s.csvLoadResult);

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [processedLoading, setProcessedLoading] = useState(false);
  const [processedError, setProcessedError] = useState('');

  async function handlePickCSV() {
    const path = await OpenCSVFileDialog();
    if (!path) return;
    setLoading(true);
    setError('');
    try {
      const result = await LoadCSVFile(yearLabel, path);
      if (result.ok) {
        setCsvLoadResult(result, path);
      } else {
        setError(result.error || 'ファイルの読み込みに失敗しました');
      }
    } catch (e: any) {
      setError(String(e));
    } finally {
      setLoading(false);
    }
  }

  async function handleLoadProcessed() {
    const path = await OpenDialog();
    if (!path) return;
    setProcessedLoading(true);
    setProcessedError('');
    try {
      const result = await LoadProcessedCSV(yearLabel, path);
      if (result.ok) {
        setStep(5);
      } else {
        setProcessedError(result.error || '読み込みに失敗しました');
      }
    } catch (e: any) {
      setProcessedError(String(e));
    } finally {
      setProcessedLoading(false);
    }
  }

  return (
    <div>
      <h3>ステップ1: CSVファイルのアップロード</h3>
      <p>
        ストレスチェック結果（CSV形式）を読み込んで分析可能な形式に変換します。
        変換対象のCSVファイルを選択してください。
      </p>
      <p>すでに処理済みのCSVがある場合は、下の「処理済みCSVを読み込む」をご利用ください。</p>

      <button onClick={handlePickCSV} disabled={loading} className="btn-primary">
        {loading ? '読み込み中...' : 'CSVファイルを選択'}
      </button>

      {error && <p style={{ color: 'red', marginTop: 8 }}>{error}</p>}

      {csvLoadResult && (
        <div style={{ marginTop: 12 }}>
          <p style={{ color: 'green' }}>
            ✓ {csvLoadResult.fileName} 読み込み成功: {csvLoadResult.rowCount} 行 ×{' '}
            {csvLoadResult.colCount} 列
          </p>

          <div style={{ overflowX: 'auto', border: '1px solid #ddd', borderRadius: 4 }}>
            <table style={{ borderCollapse: 'collapse', width: '100%', fontSize: 13 }}>
              <thead>
                <tr style={{ background: '#f5f5f5' }}>
                  {csvLoadResult.headers.map((h, i) => (
                    <th
                      key={i}
                      style={{ padding: '4px 8px', borderBottom: '1px solid #ddd', textAlign: 'left', whiteSpace: 'nowrap' }}
                    >
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {csvLoadResult.preview.map((row, ri) => (
                  <tr key={ri} style={{ background: ri % 2 === 0 ? '#fff' : '#fafafa' }}>
                    {row.map((cell, ci) => (
                      <td
                        key={ci}
                        style={{ padding: '4px 8px', borderBottom: '1px solid #eee', whiteSpace: 'nowrap' }}
                      >
                        {cell}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div style={{ marginTop: 16 }}>
            <button
              className="btn-primary"
              onClick={() => setStep(2)}
            >
              次へ：列名マッピング →
            </button>
          </div>
        </div>
      )}

      <hr style={{ marginTop: 24 }} />
      <h3>処理済みCSVを読み込む</h3>
      <p>過去に変換・保存した処理済みCSVファイルがある場合はこちらから読み込めます。</p>
      <button onClick={handleLoadProcessed} disabled={processedLoading} className="btn-primary">
        {processedLoading ? '読み込み中...' : '処理済みCSVを選択'}
      </button>
      {processedError && <p style={{ color: 'red', marginTop: 8 }}>{processedError}</p>}
    </div>
  );
}
