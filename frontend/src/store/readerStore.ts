import { create } from "zustand";

interface ReaderSelection {
  raw: string;
  cleaned: string;
}

interface ReaderSnapshot {
  dataUrl: string;
  width: number;
  height: number;
}

interface FigureCapture {
  id: string;
  itemId: string;
  page: number;
  title: string;
  dataUrl: string;
  width: number;
  height: number;
  createdAt: string;
}

interface ReaderState {
  selection: ReaderSelection;
  snapshot: ReaderSnapshot | null;
  activePage: number;
  desiredPage: number | null;
  anchorText: string;
  figureCaptures: FigureCapture[];
  setSelection: (selection: ReaderSelection) => void;
  clearSelection: () => void;
  setSnapshot: (snapshot: ReaderSnapshot | null) => void;
  setActivePage: (page: number) => void;
  jumpToPage: (page: number) => void;
  navigateToAnchor: (page: number, anchorText: string) => void;
  clearAnchor: () => void;
  addFigureCapture: (capture: FigureCapture) => void;
  addFigureCaptures: (captures: FigureCapture[]) => void;
}

export const useReaderStore = create<ReaderState>((set) => ({
  selection: { raw: "", cleaned: "" },
  snapshot: null,
  activePage: 1,
  desiredPage: null,
  anchorText: "",
  figureCaptures: [],
  setSelection(selection) {
    set({ selection });
  },
  clearSelection() {
    set({ selection: { raw: "", cleaned: "" } });
  },
  setSnapshot(snapshot) {
    set({ snapshot });
  },
  setActivePage(page) {
    set({ activePage: page });
  },
  jumpToPage(page) {
    set({ desiredPage: page, anchorText: "" });
  },
  navigateToAnchor(page, anchorText) {
    set({ desiredPage: page, anchorText });
  },
  clearAnchor() {
    set({ desiredPage: null, anchorText: "" });
  },
  addFigureCapture(capture) {
    set((state) => ({ figureCaptures: [capture, ...state.figureCaptures] }));
  },
  addFigureCaptures(captures) {
    set((state) => ({
      figureCaptures: [...captures, ...state.figureCaptures],
    }));
  },
}));
