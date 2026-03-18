import { useState } from 'react';
import {
  ApplyMappingAndCalculate,
  LoadValueMapConfig,
  SaveValueMapConfig,
  OpenJSONFileDialog,
  SaveFileDialog,
  DefaultConfigSaveFileName,
} from '../../../wailsjs/go/main/App';
import { useWizardStore, SECTIONS } from '../../store/wizardStore';

// Question texts (q1-q80) - a subset of key questions for display
const Q_TEXTS: string[] = [
  '非常にたくさんの仕事をしなければならない', // 1
  '時間内に仕事が処理しきれない', // 2
  '一生懸命働かなければならない', // 3
  'かなり注意を集中する必要がある', // 4
  '高度の知識や技術が必要なむずかしい仕事だ', // 5
  '勤務時間中はいつも仕事のことを考えていなければならない', // 6
  'からだを大変よく使う仕事だ', // 7
  '自分のペースで仕事ができる', // 8
  '自分で仕事の順番・やり方を決めることができる', // 9
  '職場の仕事の方針に自分の意見を反映できる', // 10
  '自分の技能や知識を仕事で使うことができる', // 11
  '私の部署内で意見のくい違いがある', // 12
  '私の部署と他の部署とはうまが合わない', // 13
  '私の職場の雰囲気は友好的である', // 14
  '私の職場の作業環境（騒音、照明、温度、換気など）はよくない', // 15
  '仕事の内容は自分にあっている', // 16
  '働きがいのある仕事だ', // 17
  '気分が沈み込んで、何が起こっても気が晴れない気がする', // 18
  '怒りを感じ、すぐにいらいらさせられる', // 19
  '消耗した、疲れ果てたと感じる', // 20
  '神経過敏に感じる', // 21
  'ひどく疲れた', // 22
  '頭が痛い', // 23
  '首が痛い', // 24
  '背中が痛い', // 25
  '肩こりがある', // 26
  '腰が痛い', // 27
  '目が疲れる', // 28
  '動悸や息切れがある', // 29
  '胃腸の具合が悪い', // 30
  '食欲がない', // 31
  'めまいがある', // 32
  '身体のあちこちが痛む', // 33
  'よく眠れない', // 34
  '心配事があってよく眠れない', // 35
  'くよくよ考えて気持ちが重い', // 36
  '気持ちが落ち着かない', // 37
  '活気がわいてくる', // 38
  'いきいきする', // 39
  'びくびくしたり怖気づくことがある', // 40
  '怒りを感じる', // 41
  '内心腹立たしい', // 42
  'イライラしている', // 43
  'ひどく疲れた感じがする', // 44
  'へとへとだ', // 45
  'だるい', // 46
  '職場の同僚と比べ、私の仕事上の能力は劣っている', // 47
  '私は不快な思いをさせられている', // 48
  '私の感情や気持ちに対する配慮がされている', // 49
  '困難なとき，頼りになる上司や同僚がいる', // 50
  '個人的な問題を相談できる人が職場にいる', // 51
  '配偶者（パートナー）、家族、友人等から支援が受けられる', // 52
  'あなたの職場での人間関係について、この1年間でもっとも辛かったことは何ですか？', // 53
  'あなたの職場での作業環境について、この1年間でもっとも辛かったことは何ですか？', // 54
  'あなたが今担っている役割や仕事内容について', // 55
  '仕事に満足だ', // 56
  '家庭生活に満足だ', // 57
  '働く上で、職場での人間関係がうまくいっている', // 58
  '仕事の内容は自分にあっている（再）', // 59
  '職場での自分の役割・権限範囲は明確だ', // 60
  '職場での自分の役割・権限範囲は明確だ（再）', // 61
  '職場での自分の役割・権限範囲は明確だ（再2）', // 62
  '職場での自分の役割・権限範囲は明確だ（再3）', // 63
  '職場での自分の役割・権限範囲は明確だ（再4）', // 64
  '職場での自分の役割・権限範囲は明確だ（再5）', // 65
  '職場での自分の役割・権限範囲は明確だ（再6）', // 66
  '職場での自分の役割・権限範囲は明確だ（再7）', // 67
  '職場での自分の役割・権限範囲は明確だ（再8）', // 68
  '職場での自分の役割・権限範囲は明確だ（再9）', // 69
  '職場での自分の役割・権限範囲は明確だ（再10）', // 70
  '職場での自分の役割・権限範囲は明確だ（再11）', // 71
  '職場での自分の役割・権限範囲は明確だ（再12）', // 72
  '職場での自分の役割・権限範囲は明確だ（再13）', // 73
  '職場での自分の役割・権限範囲は明確だ（再14）', // 74
  '職場での自分の役割・権限範囲は明確だ（再15）', // 75
  '職場での自分の役割・権限範囲は明確だ（再16）', // 76
  '職場での自分の役割・権限範囲は明確だ（再17）', // 77
  '職場での自分の役割・権限範囲は明確だ（再18）', // 78
  '職場での自分の役割・権限範囲は明確だ（再19）', // 79
  '職場での自分の役割・権限範囲は明確だ（再20）', // 80
];

export function Step3ValueMap() {
  const {
    yearLabel,
    csvLoadResult,
    colMap,
    valMap,
    setValMap,
    updateValMapGender,
    updateValMapIndividual,
    applyBulkSection,
    setApplyResult,
    setStep,
  } = useWizardStore();

  const [activeSection, setActiveSection] = useState<string>('bulk');
  const [applying, setApplying] = useState(false);
  const [applyError, setApplyError] = useState('');

  // All unique values found in the CSV
  const uniqueVals = csvLoadResult?.uniqueVals ?? [];
  const valOptions = ['', ...uniqueVals];

  // Bulk section state
  const [bulkVals, setBulkVals] = useState<Record<string, string[]>>(() =>
    Object.fromEntries(SECTIONS.map((s) => [s.key, ['', '', '', '']]))
  );

  async function handleLoadConfig() {
    const { OpenJSONFileDialog: OpenDialog } = await import('../../../wailsjs/go/main/App');
    const path = await OpenDialog();
    if (!path) return;
    try {
      const cfg = await LoadValueMapConfig(path);
      setValMap(cfg);
    } catch (e) {
      alert('設定の読み込みに失敗: ' + e);
    }
  }

  async function handleSaveConfig() {
    const defaultName = await DefaultConfigSaveFileName('value_mapping', yearLabel);
    const path = await SaveFileDialog(defaultName, 'json');
    if (!path) return;
    try {
      await SaveValueMapConfig(valMap, path);
    } catch (e) {
      alert('保存に失敗: ' + e);
    }
  }

  function handleBulkApply(sectionKey: string) {
    const vals = bulkVals[sectionKey];
    if (vals.some((v) => v === '')) {
      alert('一括設定する4つの値をすべて選択してください。');
      return;
    }
    applyBulkSection(sectionKey, vals);
  }

  async function handleFinish() {
    setApplying(true);
    setApplyError('');
    try {
      const result = await ApplyMappingAndCalculate(yearLabel, colMap, valMap);
      if (result.ok) {
        setApplyResult(result);
        setStep(4);
      } else {
        setApplyError(result.error || '処理に失敗しました');
      }
    } catch (e: any) {
      setApplyError(String(e));
    } finally {
      setApplying(false);
    }
  }

  return (
    <div>
      <h3>ステップ3: 値マッピング</h3>
      <p>CSVファイル内の実際の値を分析で使用する標準値に対応付けてください。</p>

      <div style={{ display: 'flex', gap: 8, marginBottom: 16 }}>
        <button onClick={handleLoadConfig} className="btn-default">
          📂 設定ファイルを読み込む
        </button>
        <button onClick={handleSaveConfig} className="btn-default">
          💾 設定を保存
        </button>
      </div>

      {/* Gender mapping */}
      <h4>性別の値マッピング</h4>
      <div style={{ display: 'flex', gap: 16, padding: 12, background: '#f9f9f9', borderRadius: 4, marginBottom: 16 }}>
        {(['male', 'female'] as const).map((field) => (
          <label key={field} style={{ display: 'flex', flexDirection: 'column', gap: 4, fontSize: 13, flex: 1 }}>
            <span>CSV内の「{field === 'male' ? '男性' : '女性'}」に対応する値:</span>
            <select
              value={valMap.gender[field]}
              onChange={(e) => updateValMapGender(field, e.target.value)}
              style={{ padding: '4px 6px' }}
            >
              {valOptions.map((v) => (
                <option key={v} value={v}>
                  {v === '' ? '（未選択）' : v}
                </option>
              ))}
            </select>
          </label>
        ))}
      </div>

      {/* NBJSQ value mapping tabs */}
      <h4>NBJSQ の値マッピング</h4>

      {/* Tab bar */}
      <div style={{ display: 'flex', borderBottom: '2px solid #337ab7', marginBottom: 0 }}>
        {[{ key: 'bulk', label: '一括設定' }, ...SECTIONS.map((s) => ({ key: s.key, label: s.key }))].map(
          (tab) => (
            <button
              key={tab.key}
              onClick={() => setActiveSection(tab.key)}
              style={{
                padding: '6px 14px',
                border: 'none',
                background: activeSection === tab.key ? '#337ab7' : '#eee',
                color: activeSection === tab.key ? '#fff' : '#333',
                cursor: 'pointer',
                marginRight: 2,
                borderRadius: '4px 4px 0 0',
                fontSize: 13,
              }}
            >
              {tab.label}
            </button>
          )
        )}
      </div>

      {/* Tab content */}
      <div
        style={{
          border: '1px solid #ddd',
          borderTop: 'none',
          padding: 12,
          maxHeight: 480,
          overflowY: 'auto',
        }}
      >
        {activeSection === 'bulk' ? (
          <BulkTab
            sections={SECTIONS}
            bulkVals={bulkVals}
            setBulkVals={setBulkVals}
            valOptions={valOptions}
            onApply={handleBulkApply}
          />
        ) : (
          <IndividualTab
            section={SECTIONS.find((s) => s.key === activeSection)!}
            valMap={valMap.nbjsq_individual}
            valOptions={valOptions}
            onUpdate={updateValMapIndividual}
          />
        )}
      </div>

      {applyError && <p style={{ color: 'red', marginTop: 8 }}>{applyError}</p>}

      <div style={{ marginTop: 16, display: 'flex', gap: 8 }}>
        <button onClick={() => setStep(2)} className="btn-default">
          ← 戻る
        </button>
        <button onClick={handleFinish} disabled={applying} className="btn-success">
          {applying ? '処理中...' : '✓ 設定完了・スコア計算'}
        </button>
      </div>
    </div>
  );
}

// --- Sub-components ---

interface BulkTabProps {
  sections: typeof SECTIONS;
  bulkVals: Record<string, string[]>;
  setBulkVals: React.Dispatch<React.SetStateAction<Record<string, string[]>>>;
  valOptions: string[];
  onApply: (key: string) => void;
}

function BulkTab({ sections, bulkVals, setBulkVals, valOptions, onApply }: BulkTabProps) {
  return (
    <div>
      {sections.map((section) => {
        const vals = bulkVals[section.key] ?? ['', '', '', ''];
        return (
          <div key={section.key} style={{ marginBottom: 16, paddingBottom: 16, borderBottom: '1px solid #eee' }}>
            <strong>{section.label}</strong>
            <div style={{ display: 'flex', gap: 8, marginTop: 8, alignItems: 'flex-end' }}>
              {section.choices.map((choice, i) => (
                <label key={i} style={{ display: 'flex', flexDirection: 'column', gap: 2, fontSize: 12, flex: 1 }}>
                  <span>スコア{i + 1}: {choice}</span>
                  <select
                    value={vals[i]}
                    onChange={(e) => {
                      const newVals = [...vals];
                      newVals[i] = e.target.value;
                      setBulkVals((prev) => ({ ...prev, [section.key]: newVals }));
                    }}
                    style={{ padding: '3px 4px', fontSize: 12 }}
                  >
                    {valOptions.map((v) => (
                      <option key={v} value={v}>
                        {v === '' ? '未選択' : v}
                      </option>
                    ))}
                  </select>
                </label>
              ))}
              <button
                onClick={() => onApply(section.key)}
                style={{ padding: '4px 10px', fontSize: 12, marginBottom: 2 }}
                className="btn-primary"
              >
                上書き適用
              </button>
            </div>
          </div>
        );
      })}
    </div>
  );
}

interface IndividualTabProps {
  section: (typeof SECTIONS)[number];
  valMap: Record<string, string[]>;
  valOptions: string[];
  onUpdate: (qKey: string, idx: number, value: string) => void;
}

function IndividualTab({ section, valMap, valOptions, onUpdate }: IndividualTabProps) {
  const [from, to] = section.qRange;
  return (
    <div>
      {/* Header row */}
      <div style={{ display: 'grid', gridTemplateColumns: '220px repeat(4, 1fr)', gap: 4, fontSize: 12, fontWeight: 'bold', marginBottom: 4 }}>
        <div>質問</div>
        {section.choices.map((c, i) => (
          <div key={i}>スコア{i + 1}: {c}</div>
        ))}
      </div>
      {Array.from({ length: to - from + 1 }, (_, i) => {
        const qNum = from + i;
        const qKey = `q${qNum}`;
        const vals = valMap[qKey] ?? ['', '', '', ''];
        return (
          <div
            key={qKey}
            style={{
              display: 'grid',
              gridTemplateColumns: '220px repeat(4, 1fr)',
              gap: 4,
              marginBottom: 4,
              alignItems: 'center',
            }}
          >
            <div style={{ fontSize: 12, color: '#555' }}>
              <strong>Q{qNum}</strong> {Q_TEXTS[qNum - 1]?.substring(0, 28)}
            </div>
            {[0, 1, 2, 3].map((idx) => (
              <select
                key={idx}
                value={vals[idx] ?? ''}
                onChange={(e) => onUpdate(qKey, idx, e.target.value)}
                style={{ fontSize: 12, padding: '2px 4px' }}
              >
                {valOptions.map((v) => (
                  <option key={v} value={v}>
                    {v === '' ? '未選択' : v}
                  </option>
                ))}
              </select>
            ))}
          </div>
        );
      })}
    </div>
  );
}
