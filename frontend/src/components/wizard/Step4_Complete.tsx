import { useState } from 'react';
import { SaveProcessedCSV, SaveFileDialog, DefaultSaveFileName } from '../../../wailsjs/go/main/App';
import { useWizardStore } from '../../store/wizardStore';

export function Step4Complete() {
  const { yearLabel, applyResult } = useWizardStore();
  const [saving, setSaving] = useState(false);
  const [saveMsg, setSaveMsg] = useState('');

  async function handleSave() {
    setSaving(true);
    setSaveMsg('');
    try {
      const defaultName = await DefaultSaveFileName(yearLabel);
      const path = await SaveFileDialog(defaultName, 'csv');
      if (!path) return;
      await SaveProcessedCSV(yearLabel, path);
      setSaveMsg('✓ 保存完了: ' + path);
    } catch (e: any) {
      setSaveMsg('❌ 保存エラー: ' + String(e));
    } finally {
      setSaving(false);
    }
  }

  return (
    <div>
      <h3>ステップ4: データ取り込み完了</h3>
      <p>データの取り込みと変換が完了しました。</p>

      {applyResult && (
        <div style={{ padding: 12, background: '#e8f5e9', borderRadius: 4, marginBottom: 16 }}>
          <p>✓ 処理レコード数: <strong>{applyResult.recordCount}</strong></p>
          <p>✓ 高ストレス者数: <strong>{applyResult.highStressN}</strong>（{((applyResult.highStressN / applyResult.recordCount) * 100).toFixed(1)}%）</p>
          {applyResult.incompleteN > 0 && (
            <p>⚠ 不完全回答（NA含む）: <strong>{applyResult.incompleteN}</strong></p>
          )}
        </div>
      )}

      <p>変換されたデータをCSVとして保存しておくと、次回以降は変換作業なしで再利用できます。</p>

      <button onClick={handleSave} disabled={saving} className="btn-primary">
        {saving ? '保存中...' : '💾 処理済みデータをCSVとして保存'}
      </button>

      {saveMsg && (
        <p style={{ marginTop: 8, color: saveMsg.startsWith('✓') ? 'green' : 'red' }}>{saveMsg}</p>
      )}

      <hr style={{ marginTop: 24 }} />
      <p>左のメニューから「分析」を選択してください。</p>
      <p>
        <em>2年度分のデータを取り込むと、すべての分析機能（要因探索を含む）が利用できます。</em>
      </p>
    </div>
  );
}

export function Step5Complete() {
  return (
    <div>
      <h3>データ取り込み完了</h3>
      <p>処理済みデータの読み込みが完了しました。</p>
      <p>今年度分と昨年度分の2年度分を取り込むと、すべての分析機能が利用できます。</p>
      <p>今年度分のみの場合、要因探索（ロジスティック回帰）は利用できません。</p>
    </div>
  );
}
