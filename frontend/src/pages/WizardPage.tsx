import { useState } from 'react';
import { useWizardStore } from '../store/wizardStore';
import { StepIndicator } from '../components/wizard/StepIndicator';
import { Step1Upload } from '../components/wizard/Step1_Upload';
import { Step2ColumnMap } from '../components/wizard/Step2_ColumnMap';
import { Step3ValueMap } from '../components/wizard/Step3_ValueMap';
import { Step4Complete, Step5Complete } from '../components/wizard/Step4_Complete';

const YEAR_LABELS = ['今年度', '昨年度'];

export function WizardPage() {
  const { step, yearLabel, setYearLabel, reset } = useWizardStore();
  const [showConfirm, setShowConfirm] = useState(false);

  function handleYearChange(newLabel: string) {
    if (newLabel === yearLabel) return;
    // If already in progress, confirm before switching
    if (step > 1) {
      if (!window.confirm(`${yearLabel}の設定を破棄して${newLabel}に切り替えますか？`)) return;
    }
    reset();
    setYearLabel(newLabel);
  }

  return (
    <div style={{ padding: '16px 24px', maxWidth: 1200, margin: '0 auto' }}>
      {/* Year tabs */}
      <div style={{ display: 'flex', gap: 4, marginBottom: 16 }}>
        {YEAR_LABELS.map((label) => (
          <button
            key={label}
            onClick={() => handleYearChange(label)}
            style={{
              padding: '8px 20px',
              border: 'none',
              borderRadius: '4px 4px 0 0',
              background: yearLabel === label ? '#337ab7' : '#ddd',
              color: yearLabel === label ? '#fff' : '#333',
              cursor: 'pointer',
              fontWeight: yearLabel === label ? 'bold' : 'normal',
            }}
          >
            {label}
          </button>
        ))}
      </div>

      {/* Step indicator */}
      <StepIndicator step={step} yearLabel={yearLabel} />
      <hr />

      {/* Step content */}
      {step === 1 && <Step1Upload />}
      {step === 2 && <Step2ColumnMap />}
      {step === 3 && <Step3ValueMap />}
      {step === 4 && <Step4Complete />}
      {step === 5 && <Step5Complete />}
    </div>
  );
}
