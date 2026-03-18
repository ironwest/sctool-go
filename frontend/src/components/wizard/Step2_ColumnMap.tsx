import { useEffect, useState } from 'react';
import {
  AutoDetectColumns,
  LoadColumnMapConfig,
  SaveColumnMapConfig,
  OpenJSONFileDialog,
  SaveFileDialog,
  DefaultConfigSaveFileName,
} from '../../../wailsjs/go/main/App';
import { useWizardStore } from '../../store/wizardStore';
import { main } from '../../../wailsjs/go/models';
// eslint-disable-next-line @typescript-eslint/no-unused-vars
type _BasicAttributesMap = main.BasicAttributesMap;

const BASIC_FIELDS: { key: keyof main.BasicAttributesMap; label: string }[] = [
  { key: 'empid', label: '社員番号' },
  { key: 'age', label: '年齢' },
  { key: 'gender', label: '性別' },
  { key: 'dept1', label: '部署（大分類）' },
  { key: 'dept2', label: '部署（中分類）' },
];

export function Step2ColumnMap() {
  const { yearLabel, csvLoadResult, colMap, setColMap, updateColMapQ, updateColMapBasic, setStep } =
    useWizardStore();
  const [autoDetecting, setAutoDetecting] = useState(false);

  const headers = csvLoadResult?.headers ?? [];
  const headerOptions = ['', ...headers];

  // Auto-detect on mount
  useEffect(() => {
    handleAutoDetect();
  }, []);

  async function handleAutoDetect() {
    if (!csvLoadResult) return;
    setAutoDetecting(true);
    try {
      const result = await AutoDetectColumns(yearLabel);
      setColMap(main.ColumnMapConfig.createFrom({
        ...colMap,
        nbjsq_questions: result.nbjsq_questions,
      }));
    } finally {
      setAutoDetecting(false);
    }
  }

  async function handleLoadConfig() {
    const path = await OpenJSONFileDialog();
    if (!path) return;
    try {
      const cfg = await LoadColumnMapConfig(path);
      setColMap(cfg);
    } catch (e) {
      alert('設定ファイルの読み込みに失敗しました: ' + e);
    }
  }

  async function handleSaveConfig() {
    const defaultName = await DefaultConfigSaveFileName('column_mapping', yearLabel);
    const path = await SaveFileDialog(defaultName, 'json');
    if (!path) return;
    try {
      await SaveColumnMapConfig(colMap, path);
    } catch (e) {
      alert('設定の保存に失敗しました: ' + e);
    }
  }

  return (
    <div>
      <h3>ステップ2: 列名マッピング</h3>
      <p>アップロードされたCSVの列名を、分析で使用する標準項目に対応付けてください。</p>

      <div style={{ display: 'flex', gap: 8, marginBottom: 16 }}>
        <button onClick={handleLoadConfig} className="btn-default">
          📂 設定ファイルを読み込む (.json)
        </button>
        <button onClick={handleSaveConfig} className="btn-default">
          💾 現在の設定を保存
        </button>
        <button onClick={handleAutoDetect} disabled={autoDetecting} className="btn-default">
          🔍 {autoDetecting ? '自動検出中...' : '列を自動検出'}
        </button>
      </div>

      {/* Basic attributes */}
      <h4>基本属性列の設定</h4>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 8, marginBottom: 16, padding: 12, background: '#f9f9f9', borderRadius: 4 }}>
        {BASIC_FIELDS.map(({ key, label }) => (
          <label key={key} style={{ display: 'flex', flexDirection: 'column', gap: 4, fontSize: 13 }}>
            <span>{label}</span>
            <select
              value={colMap.basic_attributes[key] ?? ''}
              onChange={(e) => updateColMapBasic(key, e.target.value)}
              style={{ padding: '4px 6px' }}
            >
              {headerOptions.map((h) => (
                <option key={h} value={h}>
                  {h === '' ? '（未選択）' : h}
                </option>
              ))}
            </select>
          </label>
        ))}
      </div>

      {/* Question columns */}
      <h4>NBJSQ 質問項目の列設定</h4>
      <div
        style={{
          maxHeight: 520,
          overflowY: 'auto',
          border: '1px solid #ddd',
          padding: 12,
          borderRadius: 4,
        }}
      >
        {Array.from({ length: 20 }, (_, rowIdx) => {
          const base = rowIdx * 4;
          return (
            <div key={rowIdx} style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 8, marginBottom: 6 }}>
              {[0, 1, 2, 3].map((offset) => {
                const qNum = base + offset + 1;
                if (qNum > 80) return null;
                const qKey = `q${qNum}`;
                return (
                  <label key={qKey} style={{ display: 'flex', flexDirection: 'column', gap: 2, fontSize: 12 }}>
                    <span style={{ color: '#555' }}>質問 {qNum}</span>
                    <select
                      value={colMap.nbjsq_questions[qKey] ?? ''}
                      onChange={(e) => updateColMapQ(qKey, e.target.value)}
                      style={{ padding: '3px 4px', fontSize: 12 }}
                    >
                      {headerOptions.map((h) => (
                        <option key={h} value={h}>
                          {h === '' ? '未選択' : h}
                        </option>
                      ))}
                    </select>
                  </label>
                );
              })}
            </div>
          );
        })}
      </div>

      <div style={{ marginTop: 16, display: 'flex', gap: 8 }}>
        <button onClick={() => setStep(1)} className="btn-default">
          ← 戻る
        </button>
        <button onClick={() => setStep(3)} className="btn-primary">
          次へ：値マッピング →
        </button>
      </div>
    </div>
  );
}
