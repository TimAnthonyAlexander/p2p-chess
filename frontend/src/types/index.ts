export interface User {
  id?: string;
  handle: string;
  email?: string;
  token?: string;
}

export interface AuthState {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  loading: boolean;
  error: string | null;
}

export interface TimeControl {
  value: string;
  label: string;
}

export interface MatchConfig {
  tc: string; // time control like "5+0"
  rated: boolean;
}

export interface ICEServer {
  urls: string;
  username?: string;
  credential?: string;
}

export interface MatchDetails {
  matchId: string;
  matchKey: string;
  color: 'white' | 'black';
  joinToken: string;
  turn: {
    url: string;
    username: string;
    password: string;
  };
  iceServers?: ICEServer[];
  sdpOffer?: RTCSessionDescriptionInit | null;
}

export interface GameState {
  fen: string;
  matchId: string | null;
  matchKey: string | null;
  joinToken: string | null;
  color: 'white' | 'black' | null;
  isMyTurn: boolean;
  opponentHandle: string | null;
  whiteTime: number;
  blackTime: number;
  result: GameResult | null;
  moves: string[];
}

export type GameResult = {
  winner: 'white' | 'black' | 'draw';
  method: 'checkmate' | 'stalemate' | 'timeout' | 'resignation' | 'agreement';
};

export interface WebRTCMessage {
  type: 'move' | 'resign' | 'draw_offer' | 'draw_accept' | 'draw_decline';
  move?: string;
  fen?: string;
}

export interface APIResponse<T = any> {
  data?: T;
  error?: string;
}
