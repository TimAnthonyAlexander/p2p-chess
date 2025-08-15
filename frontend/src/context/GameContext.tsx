import { createContext, useContext, ReactNode, useState } from 'react';
import { Chess } from 'chess.js';
import { GameState, MatchDetails, WebRTCMessage } from '../types/index.ts';
import WebRTCService from '../services/webrtc';
import { findMatch, generateHMAC, submitMove } from '../services/api';

// Define the initial state for a chess game
const initialGameState: GameState = {
  fen: 'start',  // standard starting position
  matchId: null,
  matchKey: null,
  joinToken: null,
  color: null,
  isMyTurn: false,
  opponentHandle: null,
  whiteTime: 0,
  blackTime: 0,
  result: null,
  moves: [],
};

interface GameContextType {
  gameState: GameState;
  chess: Chess | null;
  connectionState: RTCPeerConnectionState | null;
  startMatchmaking: (tc: string, rated: boolean) => Promise<boolean>;
  makeMove: (from: string, to: string, promotion?: string) => Promise<boolean>;
  resign: () => void;
  reset: () => void;
}

const GameContext = createContext<GameContextType | undefined>(undefined);

export const useGame = () => {
  const context = useContext(GameContext);
  if (!context) {
    throw new Error('useGame must be used within a GameProvider');
  }
  return context;
};

interface GameProviderProps {
  children: ReactNode;
}

export const GameProvider = ({ children }: GameProviderProps) => {
  const [gameState, setGameState] = useState<GameState>(initialGameState);
  const [chess, setChess] = useState<Chess | null>(new Chess());
  const [connectionState, setConnectionState] = useState<RTCPeerConnectionState | null>(null);

  // Start matchmaking and set up the game
  const startMatchmaking = async (tc: string, rated: boolean): Promise<boolean> => {
    try {
      // Reset game state first
      reset();
      
      // Call the API to find a match
      const result = await findMatch({ tc, rated });
      
      if (result.error || !result.data) {
        console.error('Matchmaking failed:', result.error);
        return false;
      }
      
      const matchDetails = result.data;
      
      // Initialize the chess game
      const newChess = new Chess();
      setChess(newChess);
      
      // Update game state with match details
      setGameState({
        ...initialGameState,
        fen: newChess.fen(),
        matchId: matchDetails.matchId,
        matchKey: matchDetails.matchKey,
        joinToken: matchDetails.joinToken,
        color: matchDetails.color,
        isMyTurn: matchDetails.color === 'white', // White moves first
        whiteTime: parseTimeControl(tc).baseTime,
        blackTime: parseTimeControl(tc).baseTime,
      });
      
      // Initialize WebRTC
      initializeWebRTC(matchDetails);
      
      return true;
    } catch (error) {
      console.error('Error starting matchmaking:', error);
      return false;
    }
  };
  
  // Helper to parse time control string (e.g., "5+0" into minutes and increment)
  const parseTimeControl = (tc: string) => {
    const parts = tc.split('+');
    const baseMinutes = parseInt(parts[0], 10) || 5; // Default to 5 minutes
    const increment = parseInt(parts[1], 10) || 0;   // Default to 0 increment
    
    return {
      baseTime: baseMinutes * 60, // Convert to seconds
      increment: increment,
    };
  };
  
  // Initialize WebRTC connection
  const initializeWebRTC = (matchDetails: MatchDetails) => {
    WebRTCService.initialize(
      matchDetails,
      handleWebRTCMessage,
      handleConnectionStateChange
    );
  };
  
  // Handle incoming WebRTC messages
  const handleWebRTCMessage = (message: WebRTCMessage) => {
    if (!chess) return;
    
    switch (message.type) {
      case 'move':
        if (message.move) {
          try {
            // Parse the move (should be in UCI format e.g., "e2e4")
            const fromSquare = message.move.substring(0, 2);
            const toSquare = message.move.substring(2, 4);
            const promotion = message.move.length > 4 ? message.move.substring(4, 5) : undefined;
            
            // Apply the move to the chess.js instance
            const moveResult = chess.move({
              from: fromSquare,
              to: toSquare,
              promotion: promotion as 'q' | 'r' | 'b' | 'n' | undefined,
            });
            
            if (moveResult) {
              // Update game state
              setGameState(prev => ({
                ...prev,
                fen: chess.fen(),
                isMyTurn: true, // It's now our turn
                moves: [...prev.moves, moveResult.san],
              }));
              
              // Check for game end
              checkGameEnd(chess);
            }
          } catch (err) {
            console.error('Error applying opponent move:', err);
          }
        }
        break;
        
      case 'resign':
        setGameState(prev => ({
          ...prev,
          result: {
            winner: prev.color === 'white' ? 'white' : 'black',
            method: 'resignation'
          }
        }));
        break;
        
      // Handle other message types as needed
      default:
        console.log('Unhandled message type:', message.type);
    }
  };
  
  // Handle WebRTC connection state changes
  const handleConnectionStateChange = (state: RTCPeerConnectionState) => {
    setConnectionState(state);
    
    // Handle disconnection
    if (state === 'disconnected' || state === 'failed' || state === 'closed') {
      console.log('Peer connection', state);
      // You could implement reconnection logic here
    }
  };
  
  // Make a move on the board
  const makeMove = async (
    from: string,
    to: string,
    promotion?: string
  ): Promise<boolean> => {
    if (!chess || !gameState.isMyTurn || !gameState.matchId || !gameState.matchKey) {
      return false;
    }
    
    try {
      // Attempt the move in chess.js
      const moveResult = chess.move({
        from,
        to,
        promotion: promotion as 'q' | 'r' | 'b' | 'n' | undefined
      });
      
      if (!moveResult) {
        return false; // Invalid move
      }
      
      // Format move in UCI notation for the backend
      const uciMove = `${from}${to}${promotion || ''}`;
      
      // Generate HMAC signature for move
      const signature = await generateHMAC(gameState.matchKey, uciMove);
      
      // Send move to opponent via WebRTC
      WebRTCService.sendMessage({
        type: 'move',
        move: uciMove,
        fen: chess.fen()
      });
      
      // Also submit to backend for verification and persistence
      await submitMove(gameState.matchId, uciMove, signature);
      
      // Update game state
      setGameState(prev => ({
        ...prev,
        fen: chess.fen(),
        isMyTurn: false, // It's opponent's turn now
        moves: [...prev.moves, moveResult.san],
      }));
      
      // Check for game end
      checkGameEnd(chess);
      
      return true;
    } catch (error) {
      console.error('Error making move:', error);
      return false;
    }
  };
  
  // Check if the game has ended
  const checkGameEnd = (chess: Chess) => {
    if (chess.isGameOver()) {
      let result = null;
      
      if (chess.isCheckmate()) {
        // Determine the winner based on whose turn it is (since the player who just moved caused checkmate)
        const winner = chess.turn() === 'w' ? 'black' : 'white';
        result = { winner, method: 'checkmate' as const };
      } else if (chess.isDraw()) {
        result = { winner: 'draw', method: 'stalemate' as const };
      }
      
      if (result) {
        setGameState(prev => ({
          ...prev,
          result
        }));
      }
    }
  };
  
  // Resign from the game
  const resign = () => {
    if (!gameState.matchId || gameState.result) {
      return; // Can't resign if no game or game already ended
    }
    
    // Send resign message via WebRTC
    WebRTCService.sendMessage({ type: 'resign' });
    
    // Update local game state
    setGameState(prev => ({
      ...prev,
      result: {
        winner: prev.color === 'white' ? 'black' : 'white',
        method: 'resignation'
      }
    }));
    
    // Ideally also inform the backend about the resignation
    // This could be a special move type or a separate API call
  };
  
  // Reset the game state
  const reset = () => {
    // Close any existing WebRTC connections
    WebRTCService.close();
    
    // Reset game state
    setGameState(initialGameState);
    setChess(new Chess());
    setConnectionState(null);
  };
  
  return (
    <GameContext.Provider
      value={{
        gameState,
        chess,
        connectionState,
        startMatchmaking,
        makeMove,
        resign,
        reset
      }}
    >
      {children}
    </GameContext.Provider>
  );
};

export default GameContext;
