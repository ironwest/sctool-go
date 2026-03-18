import { WizardStep } from '../../store/wizardStore';

interface Props {
  step: WizardStep;
  yearLabel: string;
}

const STEP_LABELS = ['CSVアップロード', '列名マッピング', '値マッピング', '完了'];

export function StepIndicator({ step, yearLabel }: Props) {
  const displayStep = step > 4 ? 4 : step;
  const pct = Math.round((displayStep / 4) * 100);

  return (
    <div style={{ marginBottom: 16 }}>
      <h4 style={{ margin: '0 0 8px' }}>
        {yearLabel} 設定 — ステップ {displayStep}: {STEP_LABELS[displayStep - 1]}
      </h4>
      <div style={{ background: '#eee', borderRadius: 5, height: 20 }}>
        <div
          style={{
            background: '#337ab7',
            width: `${pct}%`,
            height: '100%',
            borderRadius: 5,
            textAlign: 'center',
            color: 'white',
            lineHeight: '20px',
            fontSize: 12,
            transition: 'width 0.3s',
          }}
        >
          {pct}%
        </div>
      </div>
    </div>
  );
}
