import { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { useGame } from '../context/GameContext';
import { useAuth } from '../context/AuthContext';
import { Chessboard } from 'react-chessboard';
import type { Square } from 'chess.js';
import Button from '../components/Button';
import Card from '../components/Card';

const formatTime = (seconds: number): string => {
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return `${mins}:${secs < 10 ? '0' : ''}${secs}`;
};

const GamePage = () => {
  const { gameState, chess, connectionState, makeMove, resign, reset } = useGame();
  const { user } = useAuth();
  const navigate = useNavigate();
  
  const [promotion, setPromotion] = useState<'q' | 'r' | 'b' | 'n' | undefined>(undefined);
  const [whiteTimeDisplay, setWhiteTimeDisplay] = useState('');
  const [blackTimeDisplay, setBlackTimeDisplay] = useState('');
  const [showConfirmResign, setShowConfirmResign] = useState(false);
  
  // Format times for display
  useEffect(() => {
    setWhiteTimeDisplay(formatTime(gameState.whiteTime));
    setBlackTimeDisplay(formatTime(gameState.blackTime));
  }, [gameState.whiteTime, gameState.blackTime]);
  
  // Manage clock countdown
  useEffect(() => {
    let timer: number | null = null;
    
    // Only count down if the game is in progress
    if (!gameState.result && gameState.matchId) {
      timer = setInterval(() => {
        // Only decrement the active player's clock
        if ((gameState.color === 'white' && !gameState.isMyTurn) || 
            (gameState.color === 'black' && gameState.isMyTurn)) {
          // It's white's turn
          setWhiteTimeDisplay(prev => {
            const parts = prev.split(':');
            let mins = parseInt(parts[0]);
            let secs = parseInt(parts[1]);
            
            if (secs > 0) {
              secs--;
            } else if (mins > 0) {
              mins--;
              secs = 59;
            }
            
            return `${mins}:${secs < 10 ? '0' : ''}${secs}`;
          });
        } else if ((gameState.color === 'black' && !gameState.isMyTurn) || 
                   (gameState.color === 'white' && gameState.isMyTurn)) {
          // It's black's turn
          setBlackTimeDisplay(prev => {
            const parts = prev.split(':');
            let mins = parseInt(parts[0]);
            let secs = parseInt(parts[1]);
            
            if (secs > 0) {
              secs--;
            } else if (mins > 0) {
              mins--;
              secs = 59;
            }
            
            return `${mins}:${secs < 10 ? '0' : ''}${secs}`;
          });
        }
      }, 1000);
    }
    
    return () => {
      if (timer) {
        clearInterval(timer);
      }
    };
  }, [gameState.color, gameState.isMyTurn, gameState.result, gameState.matchId]);
  
  // Handle piece movement
  const onDrop = useCallback(
    (sourceSquare: Square, targetSquare: Square) => {
      // Check if it's our turn
      if (!gameState.isMyTurn || gameState.result) {
        return false;
      }
      
      // Check if the move is valid
      if (!chess) return false;
      
      // Get possible moves from this square
      const moves = chess.moves({
        square: sourceSquare as Square,
        verbose: true
      });
      
      // See if our target square is in the list of valid moves
      const move = moves.find(
        (m) => m.to === targetSquare && (!m.promotion || m.promotion === promotion)
      );
      
      // If we're doing a promotion, but haven't chosen the piece yet
      if (
        moves.some(m => m.to === targetSquare && m.promotion) &&
        !promotion
      ) {
        // Store the current move attempt and prompt for promotion piece
        setPromotion('q'); // Default to queen for simplicity in MVP
        return false;
      }
      
      // If the move is invalid, return false
      if (!move) return false;
      
      // Optimistically make the move on the local board
      const previousFen = chess.fen();
      chess.move(move);
      
      // Send the move async and rollback if fails
      makeMove(sourceSquare, targetSquare, promotion).then(success => {
        if (!success) {
          // Rollback if server rejects
          chess.load(previousFen);
        }
      });
      
      // Reset promotion state
      setPromotion(undefined);
      
      // Return true to apply the optimistic move immediately
      return true;
    },
    [chess, gameState.isMyTurn, gameState.result, makeMove, promotion]
  );
  
  // Handle resign confirmation
  const handleResign = () => {
    if (showConfirmResign) {
      resign();
      setShowConfirmResign(false);
    } else {
      setShowConfirmResign(true);
    }
  };
  
  // Handle return to lobby
  const handleReturnToLobby = () => {
    reset();
    navigate('/');
  };
  
  // Determine if the game has ended
  const gameEnded = gameState.result !== null;
  
  // Determine result message
  const getResultMessage = () => {
    if (!gameState.result) return null;
    
    const { winner, method } = gameState.result;
    const youWon = 
      (winner === 'white' && gameState.color === 'white') || 
      (winner === 'black' && gameState.color === 'black');
    
    if (winner === 'draw') {
      return 'Game Drawn';
    }
    
    if (youWon) {
      return `You won by ${method}`;
    }
    
    return `You lost by ${method}`;
  };
  
  // Connection status message
  const getConnectionStatus = () => {
    if (!connectionState) return 'Setting up connection...';
    
    switch (connectionState) {
      case 'new':
      case 'connecting':
        return 'Connecting to opponent...';
      case 'connected':
        return 'Connected to opponent';
      case 'disconnected':
      case 'closed':
        return 'Disconnected from opponent';
      case 'failed':
        return 'Connection failed';
      default:
        return 'Unknown connection state';
    }
  };
  
  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="max-w-6xl mx-auto px-4">
        <div className="grid lg:grid-cols-3 gap-6">
          {/* Left sidebar - Player info and clocks */}
          <div className="lg:col-span-1 space-y-4">
            <Card title="Game Info">
              <div className="space-y-4">
                <div className={`flex justify-between items-center p-3 rounded-md ${gameState.isMyTurn && gameState.color === 'white' ? 'bg-blue-100' : 'bg-gray-100'}`}>
                  <div>
                    <p className="font-medium">White</p>
                    <p className="text-sm text-gray-600">
                      {gameState.color === 'white' ? user?.handle : gameState.opponentHandle || 'Opponent'}
                    </p>
                  </div>
                  <div className="text-xl font-mono">{whiteTimeDisplay}</div>
                </div>
                
                <div className={`flex justify-between items-center p-3 rounded-md ${gameState.isMyTurn && gameState.color === 'black' ? 'bg-blue-100' : 'bg-gray-100'}`}>
                  <div>
                    <p className="font-medium">Black</p>
                    <p className="text-sm text-gray-600">
                      {gameState.color === 'black' ? user?.handle : gameState.opponentHandle || 'Opponent'}
                    </p>
                  </div>
                  <div className="text-xl font-mono">{blackTimeDisplay}</div>
                </div>
                
                <div className="p-3 bg-gray-100 rounded-md text-sm">
                  <p className="font-medium">Connection</p>
                  <p className={connectionState === 'connected' ? 'text-green-600' : 'text-amber-600'}>
                    {getConnectionStatus()}
                  </p>
                </div>
                
                <div className="flex space-x-2">
                  <Button 
                    variant="danger" 
                    onClick={handleResign} 
                    disabled={gameEnded}
                    fullWidth
                  >
                    {showConfirmResign ? 'Confirm Resign' : 'Resign'}
                  </Button>
                  
                  {gameEnded && (
                    <Button 
                      variant="primary" 
                      onClick={handleReturnToLobby}
                      fullWidth
                    >
                      Return to Lobby
                    </Button>
                  )}
                </div>
                
                {showConfirmResign && !gameEnded && (
                  <p className="text-sm text-center text-red-600">
                    Are you sure you want to resign? Click the button again to confirm.
                  </p>
                )}
              </div>
            </Card>
          </div>
          
          {/* Center - Chess board */}
          <div className="lg:col-span-2">
            <Card padded={false}>
              <div className="relative">
                <Chessboard
                  position={gameState.fen}
                  onPieceDrop={onDrop}
                  boardOrientation={gameState.color || 'white'}
                  arePiecesDraggable={gameState.isMyTurn && !gameEnded}
                  customBoardStyle={{
                    borderRadius: '0.375rem',
                    boxShadow: 'rgba(0, 0, 0, 0.1) 0px 10px 15px -3px, rgba(0, 0, 0, 0.05) 0px 4px 6px -2px',
                  }}
                />
                
                {gameEnded && (
                  <div className="absolute inset-0 bg-black bg-opacity-50 flex items-center justify-center">
                    <div className="bg-white p-4 rounded-md shadow-lg text-center">
                      <h3 className="text-xl font-bold text-gray-900">Game Over</h3>
                      <p className="mt-2 text-lg">{getResultMessage()}</p>
                    </div>
                  </div>
                )}
                
                {!gameState.matchId && (
                  <div className="absolute inset-0 bg-black bg-opacity-50 flex items-center justify-center">
                    <div className="bg-white p-4 rounded-md shadow-lg text-center">
                      <h3 className="text-xl font-bold text-gray-900">No Active Game</h3>
                      <p className="mt-2">Return to the lobby to find a match</p>
                      <Button 
                        variant="primary" 
                        className="mt-4"
                        onClick={handleReturnToLobby}
                      >
                        Go to Lobby
                      </Button>
                    </div>
                  </div>
                )}
              </div>
            </Card>
            
            {/* Move history */}
            {gameState.moves.length > 0 && (
              <Card className="mt-4" title="Move History">
                <div className="grid grid-cols-2 gap-2">
                  {gameState.moves.map((move, i) => (
                    <div 
                      key={i} 
                      className="text-sm p-1 border-b"
                    >
                      {Math.floor(i / 2) + 1}.{i % 2 === 0 ? '' : '..'} {move}
                    </div>
                  ))}
                </div>
              </Card>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default GamePage;
