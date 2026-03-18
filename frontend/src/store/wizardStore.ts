import { create } from 'zustand';
import { main } from '../../wailsjs/go/models';

export type WizardStep = 1 | 2 | 3 | 4 | 5;

// Section definitions for value mapping
export const SECTIONS = [
  {
    key: 'A' as const,
    label: 'A) あなたの仕事についてうかがいます',
    qRange: [1, 17] as [number, number],
    choices: ['そうだ', 'まあそうだ', 'ややちがう', 'ちがう'],
    bulkKey: 'group_aefgh' as const,
  },
  {
    key: 'B' as const,
    label: 'B) 最近1か月間のあなたの状態についてうかがいます',
    qRange: [18, 46] as [number, number],
    choices: ['ほとんどなかった', 'ときどきあった', 'しばしばあった', 'ほとんどいつもあった'],
    bulkKey: 'group_b' as const,
  },
  {
    key: 'C' as const,
    label: 'C) あなたの周りの方々についてうかがいます',
    qRange: [47, 55] as [number, number],
    choices: ['非常に', 'かなり', '多少', 'まったくない'],
    bulkKey: 'group_c' as const,
  },
  {
    key: 'D' as const,
    label: 'D) 満足度についてうかがいます',
    qRange: [56, 57] as [number, number],
    choices: ['満足', 'まあ満足', 'やや不満足', '不満足'],
    bulkKey: 'group_d' as const,
  },
  {
    key: 'EH' as const,
    label: 'E-H) 仕事・職場・会社について',
    qRange: [58, 80] as [number, number],
    choices: ['そうだ', 'まあそうだ', 'ややちがう', 'ちがう'],
    bulkKey: 'group_aefgh' as const,
  },
] as const;

// Default empty column map
const defaultColMap = (): main.ColumnMapConfig =>
  main.ColumnMapConfig.createFrom({
    basic_attributes: { empid: '', age: '', gender: '', dept1: '', dept2: '' },
    nbjsq_questions: Object.fromEntries(
      Array.from({ length: 80 }, (_, i) => [`q${i + 1}`, ''])
    ),
  });

// Default empty value map
const defaultValMap = (): main.ValueMapConfig =>
  main.ValueMapConfig.createFrom({
    gender: { male: '', female: '' },
    nbjsq_bulk: { group_aefgh: ['', '', '', ''], group_b: ['', '', '', ''], group_c: ['', '', '', ''], group_d: ['', '', '', ''] },
    nbjsq_individual: Object.fromEntries(
      Array.from({ length: 80 }, (_, i) => [`q${i + 1}`, ['', '', '', '']])
    ),
  });

interface WizardState {
  step: WizardStep;
  yearLabel: string; // "今年度" or "昨年度"

  // Step 1 state
  csvLoadResult: main.CSVLoadResult | null;
  csvFilePath: string;

  // Step 2 state
  colMap: main.ColumnMapConfig;

  // Step 3 state
  valMap: main.ValueMapConfig;

  // Step 4 state
  applyResult: main.ApplyResult | null;

  // Actions
  setStep: (step: WizardStep) => void;
  setYearLabel: (label: string) => void;
  setCsvLoadResult: (result: main.CSVLoadResult, path: string) => void;
  setColMap: (cfg: main.ColumnMapConfig) => void;
  updateColMapQ: (qKey: string, colName: string) => void;
  updateColMapBasic: (field: keyof main.BasicAttributesMap, colName: string) => void;
  setValMap: (cfg: main.ValueMapConfig) => void;
  updateValMapGender: (field: 'male' | 'female', value: string) => void;
  updateValMapIndividual: (qKey: string, idx: number, value: string) => void;
  applyBulkSection: (sectionKey: string, vals: string[]) => void;
  setApplyResult: (result: main.ApplyResult) => void;
  reset: () => void;
}

export const useWizardStore = create<WizardState>((set, get) => ({
  step: 1,
  yearLabel: '今年度',
  csvLoadResult: null,
  csvFilePath: '',
  colMap: defaultColMap(),
  valMap: defaultValMap(),
  applyResult: null,

  setStep: (step) => set({ step }),
  setYearLabel: (label) => set({ yearLabel: label }),

  setCsvLoadResult: (result, path) => set({ csvLoadResult: result, csvFilePath: path }),

  setColMap: (cfg) => set({ colMap: cfg }),

  updateColMapQ: (qKey, colName) =>
    set((state) => ({
      colMap: main.ColumnMapConfig.createFrom({
        ...state.colMap,
        nbjsq_questions: { ...state.colMap.nbjsq_questions, [qKey]: colName },
      }),
    })),

  updateColMapBasic: (field, colName) =>
    set((state) => ({
      colMap: main.ColumnMapConfig.createFrom({
        ...state.colMap,
        basic_attributes: { ...state.colMap.basic_attributes, [field]: colName },
      }),
    })),

  setValMap: (cfg) => set({ valMap: cfg }),

  updateValMapGender: (field, value) =>
    set((state) => ({
      valMap: main.ValueMapConfig.createFrom({
        ...state.valMap,
        gender: { ...state.valMap.gender, [field]: value },
      }),
    })),

  updateValMapIndividual: (qKey, idx, value) =>
    set((state) => {
      const current = [...(state.valMap.nbjsq_individual[qKey] ?? ['', '', '', ''])];
      current[idx] = value;
      return {
        valMap: main.ValueMapConfig.createFrom({
          ...state.valMap,
          nbjsq_individual: { ...state.valMap.nbjsq_individual, [qKey]: current },
        }),
      };
    }),

  applyBulkSection: (sectionKey, vals) => {
    const section = SECTIONS.find((s) => s.key === sectionKey);
    if (!section) return;
    const [from, to] = section.qRange;
    set((state) => {
      const individual = { ...state.valMap.nbjsq_individual };
      for (let q = from; q <= to; q++) {
        individual[`q${q}`] = [...vals];
      }
      return {
        valMap: main.ValueMapConfig.createFrom({
          ...state.valMap,
          nbjsq_individual: individual,
        }),
      };
    });
  },

  setApplyResult: (result) => set({ applyResult: result }),

  reset: () =>
    set({
      step: 1,
      csvLoadResult: null,
      csvFilePath: '',
      colMap: defaultColMap(),
      valMap: defaultValMap(),
      applyResult: null,
    }),
}));
