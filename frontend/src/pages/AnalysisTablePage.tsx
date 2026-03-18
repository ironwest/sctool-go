import { useState, useEffect, useRef } from 'react';
import {
  useReactTable,
  getCoreRowModel,
  flexRender,
  createColumnHelper,
  ColumnDef,
} from '@tanstack/react-table';
import { GetAnalysisTable } from '../../wailsjs/go/main/App';
import { main, score } from '../../wailsjs/go/models';

// ---------- Column layout constants (matching setting_hensati_hyou.R) ----------

const COLUMN_GROUPS = [
  {
    header: 'アウトカム',
    color: '#dbeafe',
    cols: ['ソーシャル・キャピタル', 'ワークエンゲイジメント', '職場のハラスメント', '心理的ストレス反応合計', '仕事の負担合計'],
  },
  {
    header: '資源',
    color: '#cffafe',
    cols: ['作業レベル資源合計', '部署レベル資源合計', '事業場レベル資源'],
  },
  {
    header: '心理的ストレス反応',
    color: '#ede9fe',
    cols: ['活気', 'イライラ感', '疲労感', '不安感', '抑うつ感'],
  },
  {
    header: '仕事の負担',
    color: '#ffedd5',
    cols: ['仕事の量的負担', '仕事の質的負担', '身体的負担度', '職場での対人関係', '職場環境', '情緒的負担', '役割葛藤', 'WSB（－）'],
  },
  {
    header: '作業レベル資源',
    color: '#dcfce7',
    cols: ['仕事のコントロール', '技能の活用', '仕事の適正', '仕事の意義', '役割明確さ', '成長の機会'],
  },
  {
    header: '部署レベル資源',
    color: '#fce7f3',
    cols: ['上司の支援', '同僚の支援', '経済・地位報酬', '尊重報酬', '安定報酬', '上司のリーダーシップ', '上司の公正な態度', 'ほめてもらえる職場', '失敗を認める職場'],
  },
  {
    header: '事業場レベル資源',
    color: '#e0f2fe',
    cols: ['経営層との信頼関係', '変化への対応', '個人の尊重', '公正な人事評価', '多様な労働者への対応', 'キャリア形成', 'WSB（＋）'],
  },
] as const;

const GYOUSYU_OPTIONS = [
  '全産業', '製造業', '情報通信業', '運輸・郵便業', '卸売・小売業',
  '金融・保険業', '建設業', '医療・福祉', '教育・学習支援業', 'サービス業', '公務',
];

const GROUP_VAR_OPTIONS: { value: string; label: string }[] = [
  { value: 'dept1', label: '部署（大分類）' },
  { value: 'dept2', label: '部署（中分類）' },
  { value: 'dept1_dept2', label: '部署（大＋中分類）' },
  { value: 'age_kubun', label: '年齢区分' },
  { value: 'gender', label: '性別' },
];

// ---------- Cell color helper ----------

function cellStyle(value: number | undefined, isHensatiCol: boolean): React.CSSProperties {
  if (value == null || isNaN(value) || !isHensatiCol) return {};
  if (value >= 60) return { background: '#e6f5e6', color: '#006400', fontWeight: 'bold' };
  if (value >= 55) return { background: '#e6f5e6' };
  if (value <= 40) return { background: '#fde8e8', color: '#990000', fontWeight: 'bold' };
  if (value <= 45) return { background: '#fde8e8' };
  return {};
}

function fmtNum(v: number | undefined, digits = 0): string {
  if (v == null || isNaN(v)) return '-';
  return v.toFixed(digits);
}

function fmtPct(v: number | undefined): string {
  if (v == null || isNaN(v)) return '-';
  return (v * 100).toFixed(1) + '%';
}

// ---------- Component ----------

type Row = score.AnalysisTableRow;

interface Props {
  yearLabel: string;
}

export function AnalysisTablePage({ yearLabel }: Props) {
  const [groupVar, setGroupVar] = useState('dept1');
  const [longOrCross, setLongOrCross] = useState<'long' | 'cross'>('long');
  const [gyousyu, setGyousyu] = useState('全産業');
  const [limitN, setLimitN] = useState(0);
  const [rows, setRows] = useState<Row[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const tableRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    load();
  }, [yearLabel, groupVar, longOrCross, gyousyu]);

  async function load() {
    setLoading(true);
    setError('');
    try {
      const result = await GetAnalysisTable(yearLabel, groupVar, longOrCross, gyousyu);
      if (!result.ok) {
        setError(result.error ?? 'エラーが発生しました');
        setRows([]);
      } else {
        setRows(result.rows ?? []);
      }
    } catch (e) {
      setError(String(e));
    } finally {
      setLoading(false);
    }
  }

  // Filter rows by limitN
  const displayRows = limitN > 0
    ? rows.filter(r => r.n - r.incompleteN >= limitN)
    : rows;

  // --- Build columns ---
  const colHelper = createColumnHelper<Row>();

  const columns: ColumnDef<Row, any>[] = [
    // Group column
    colHelper.accessor('groupLabel', {
      id: 'groupLabel',
      header: groupVarLabel(groupVar),
      cell: info => (
        <span style={{ fontWeight: info.row.original.isTotal ? 'bold' : 'normal' }}>
          {info.getValue()}
        </span>
      ),
      size: 140,
      meta: { sticky: true },
    }),
    // Stats columns (no group header)
    colHelper.accessor('n', { header: '受検\n人数', cell: info => fmtNum(info.getValue()), size: 50 }),
    colHelper.accessor('incompleteN', { header: '不完全\n回答', cell: info => fmtNum(info.getValue()), size: 50 }),
    colHelper.accessor('highStressN', { header: '高ｽﾄﾚｽ\n人数', cell: info => fmtNum(info.getValue()), size: 55 }),
    colHelper.accessor('highStressRatio', { header: '高ｽﾄﾚｽ\n割合', cell: info => fmtPct(info.getValue()), size: 55 }),
    colHelper.accessor('totalRisk', { header: '総合\nﾘｽｸ', cell: info => fmtNum(info.getValue()), size: 50 }),
    // Hensati column groups
    ...COLUMN_GROUPS.map(group =>
      colHelper.group({
        id: group.header,
        header: () => (
          <span style={{ background: group.color, padding: '2px 6px', display: 'block', fontWeight: 'bold' }}>
            {group.header}
          </span>
        ),
        columns: group.cols.map(jpnName =>
          colHelper.accessor(row => row.hensati?.[jpnName], {
            id: jpnName,
            header: jpnName,
            cell: info => {
              const v = info.getValue() as number | undefined;
              return (
                <span style={cellStyle(v, true)}>
                  {fmtNum(v)}
                </span>
              );
            },
            size: 44,
          })
        ),
      })
    ),
  ];

  const table = useReactTable({
    data: displayRows,
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  // --- CSV download ---
  function downloadCSV() {
    const allCols: string[] = ['グループ', '受検人数', '不完全回答', '高ストレス人数', '高ストレス割合', '総合リスク'];
    COLUMN_GROUPS.forEach(g => g.cols.forEach(c => allCols.push(c)));

    const lines: string[] = [allCols.join(',')];
    for (const row of displayRows) {
      const cells = [
        row.groupLabel,
        row.n,
        row.incompleteN,
        row.highStressN,
        (row.highStressRatio * 100).toFixed(1) + '%',
        row.totalRisk,
      ];
      COLUMN_GROUPS.forEach(g =>
        g.cols.forEach(c => {
          const v = row.hensati?.[c];
          cells.push(v != null && !isNaN(v) ? v.toFixed(0) : '');
        })
      );
      lines.push(cells.map(c => `"${String(c).replace(/"/g, '""')}"`).join(','));
    }
    const blob = new Blob(['\uFEFF' + lines.join('\n')], { type: 'text/csv;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `analysis_${yearLabel}_${groupVar}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  }

  return (
    <div style={{ padding: '16px 24px' }}>
      <h3 style={{ marginBottom: 12 }}>偏差値表 — {yearLabel}</h3>

      {/* Controls */}
      <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', marginBottom: 12, alignItems: 'center' }}>
        <label style={{ display: 'flex', flexDirection: 'column', gap: 2, fontSize: 12 }}>
          グループ変数
          <select value={groupVar} onChange={e => setGroupVar(e.target.value)} style={{ padding: '4px 6px' }}>
            {GROUP_VAR_OPTIONS.map(o => (
              <option key={o.value} value={o.value}>{o.label}</option>
            ))}
          </select>
        </label>

        <label style={{ display: 'flex', flexDirection: 'column', gap: 2, fontSize: 12 }}>
          リスクスコア種別
          <select value={longOrCross} onChange={e => setLongOrCross(e.target.value as 'long' | 'cross')} style={{ padding: '4px 6px' }}>
            <option value="long">縦断（long）</option>
            <option value="cross">横断（cross）</option>
          </select>
        </label>

        <label style={{ display: 'flex', flexDirection: 'column', gap: 2, fontSize: 12 }}>
          業種
          <select value={gyousyu} onChange={e => setGyousyu(e.target.value)} style={{ padding: '4px 6px' }}>
            {GYOUSYU_OPTIONS.map(g => <option key={g} value={g}>{g}</option>)}
          </select>
        </label>

        <label style={{ display: 'flex', flexDirection: 'column', gap: 2, fontSize: 12 }}>
          最小N（完全回答）
          <input
            type="number"
            min={0}
            value={limitN}
            onChange={e => setLimitN(Number(e.target.value))}
            style={{ width: 70, padding: '4px 6px' }}
          />
        </label>

        <button onClick={load} disabled={loading} className="btn-default" style={{ alignSelf: 'flex-end' }}>
          {loading ? '計算中...' : '再計算'}
        </button>
        <button onClick={downloadCSV} disabled={loading || displayRows.length === 0} className="btn-default" style={{ alignSelf: 'flex-end' }}>
          CSV ダウンロード
        </button>
      </div>

      {error && <p style={{ color: 'red', marginBottom: 8 }}>{error}</p>}
      {loading && <p style={{ color: '#666' }}>計算中...</p>}

      {!loading && displayRows.length > 0 && (
        <div
          ref={tableRef}
          style={{ overflowX: 'auto', fontSize: 12, border: '1px solid #ddd', borderRadius: 4, maxHeight: '70vh', overflowY: 'auto' }}
        >
          <table style={{ borderCollapse: 'collapse', whiteSpace: 'nowrap' }}>
            <thead style={{ position: 'sticky', top: 0, zIndex: 2, background: '#f5f5f5' }}>
              {table.getHeaderGroups().map(hg => (
                <tr key={hg.id}>
                  {hg.headers.map(header => {
                    const isGroupHeader = header.subHeaders.length > 0;
                    const colGroup = COLUMN_GROUPS.find(g => g.header === header.id);
                    const bgColor = colGroup?.color ?? '#f5f5f5';
                    return (
                      <th
                        key={header.id}
                        colSpan={header.colSpan}
                        style={{
                          border: '1px solid #ccc',
                          padding: isGroupHeader ? '4px 6px' : '3px 4px',
                          background: isGroupHeader ? bgColor : '#f5f5f5',
                          fontWeight: 'bold',
                          textAlign: 'center',
                          verticalAlign: 'bottom',
                          writingMode: (!isGroupHeader && header.id !== 'groupLabel') ? 'vertical-rl' : 'horizontal-tb',
                          minWidth: isGroupHeader ? undefined : (header.id === 'groupLabel' ? 140 : 44),
                          fontSize: 11,
                          whiteSpace: 'pre-line',
                        }}
                      >
                        {header.isPlaceholder
                          ? null
                          : flexRender(header.column.columnDef.header, header.getContext())}
                      </th>
                    );
                  })}
                </tr>
              ))}
            </thead>
            <tbody>
              {table.getRowModel().rows.map(row => (
                <tr
                  key={row.id}
                  style={{
                    background: row.original.isTotal ? '#fffde7' : undefined,
                  }}
                >
                  {row.getVisibleCells().map(cell => {
                    const allHensatiCols = COLUMN_GROUPS.flatMap(g => [...g.cols] as string[]);
                    const isHensatiCol = allHensatiCols.includes(cell.column.id);
                    const rawVal = isHensatiCol
                      ? (cell.getValue() as number | undefined)
                      : undefined;
                    return (
                      <td
                        key={cell.id}
                        style={{
                          border: '1px solid #ddd',
                          padding: '3px 6px',
                          textAlign: cell.column.id === 'groupLabel' ? 'left' : 'center',
                          ...cellStyle(rawVal, isHensatiCol),
                        }}
                      >
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </td>
                    );
                  })}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {!loading && displayRows.length === 0 && !error && (
        <p style={{ color: '#999' }}>データがありません。「データ設定」でCSVを読み込んでください。</p>
      )}
    </div>
  );
}

function groupVarLabel(groupVar: string): string {
  return GROUP_VAR_OPTIONS.find(o => o.value === groupVar)?.label ?? groupVar;
}
