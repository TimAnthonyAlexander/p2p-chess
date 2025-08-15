/// <reference types="vite/client" />

declare module 'react-chessboard' {
  import type { Square } from 'chess.js';
  import type { CSSProperties, FC } from 'react';

  interface ChessboardProps {
    position?: string;
    onPieceDrop?: (sourceSquare: Square, targetSquare: Square) => boolean | Promise<boolean>;
    boardOrientation?: 'white' | 'black';
    arePiecesDraggable?: boolean;
    customBoardStyle?: CSSProperties;
  }

  export const Chessboard: FC<ChessboardProps>;
}
