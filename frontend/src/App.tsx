import { useState } from 'react';
import './App.css';
import { WizardPage } from './pages/WizardPage';

type NavItem = 'wizard' | 'analysis';

function App() {
  const [nav, setNav] = useState<NavItem>('wizard');

  return (
    <div style={{ display: 'flex', height: '100vh', fontFamily: 'sans-serif', fontSize: 14 }}>
      {/* Sidebar */}
      <nav
        style={{
          width: 200,
          background: '#2c3e50',
          color: '#ecf0f1',
          display: 'flex',
          flexDirection: 'column',
          padding: '16px 0',
          flexShrink: 0,
        }}
      >
        <div style={{ padding: '0 16px 16px', fontSize: 15, fontWeight: 'bold', borderBottom: '1px solid #4a6080' }}>
          ストレスチェック
          <br />
          集団分析ツール
        </div>
        <NavButton active={nav === 'wizard'} onClick={() => setNav('wizard')}>
          📁 データ設定
        </NavButton>
        <NavButton active={nav === 'analysis'} onClick={() => setNav('analysis')}>
          📊 分析（準備中）
        </NavButton>
      </nav>

      {/* Main content */}
      <main style={{ flex: 1, overflowY: 'auto', background: '#f5f5f5' }}>
        {nav === 'wizard' && <WizardPage />}
        {nav === 'analysis' && (
          <div style={{ padding: 24 }}>
            <h3>分析機能は Phase 3〜5 で実装予定です</h3>
            <p>まず「データ設定」からデータを読み込んでください。</p>
          </div>
        )}
      </main>
    </div>
  );
}

function NavButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      onClick={onClick}
      style={{
        background: active ? '#337ab7' : 'transparent',
        color: '#ecf0f1',
        border: 'none',
        padding: '10px 16px',
        textAlign: 'left',
        cursor: 'pointer',
        fontSize: 14,
        width: '100%',
      }}
    >
      {children}
    </button>
  );
}

export default App;
