import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { useGame } from '../context/GameContext';
import Card from '../components/Card';
import Button from '../components/Button';
import Select from '../components/Select';
import { TimeControl } from '../types/index.ts';

const LobbyPage = () => {
  const { user, logout } = useAuth();
  const { startMatchmaking } = useGame();
  const navigate = useNavigate();
  
  const [selectedTimeControl, setSelectedTimeControl] = useState('5+0');
  const [isRated, setIsRated] = useState(true);
  const [isSearching, setIsSearching] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Time control options
  const timeControlOptions: TimeControl[] = [
    { value: '1+0', label: '1+0 Bullet' },
    { value: '3+0', label: '3+0 Blitz' },
    { value: '3+2', label: '3+2 Blitz' },
    { value: '5+0', label: '5+0 Blitz' },
    { value: '5+3', label: '5+3 Blitz' },
    { value: '10+0', label: '10+0 Rapid' },
    { value: '10+5', label: '10+5 Rapid' },
    { value: '15+10', label: '15+10 Rapid' },
  ];

  const handleFindMatch = async () => {
    setError(null);
    setIsSearching(true);
    
    try {
      const success = await startMatchmaking(selectedTimeControl, isRated);
      
      if (success) {
        navigate('/game');
      } else {
        setError('Failed to find a match. Please try again.');
        setIsSearching(false);
      }
    } catch (err) {
      setError('An error occurred while searching for a match.');
      setIsSearching(false);
    }
  };

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="max-w-6xl mx-auto px-4">
        <header className="flex justify-between items-center mb-8">
          <h1 className="text-3xl font-bold text-gray-900">P2P Chess</h1>
          <div className="flex items-center space-x-4">
            <span className="text-gray-700">Welcome, {user?.handle}</span>
            <Button variant="outline" onClick={logout}>Sign Out</Button>
          </div>
        </header>
        
        <div className="grid md:grid-cols-2 gap-8">
          {/* Quick Play Card */}
          <Card title="Quick Play">
            <div className="space-y-6">
              <div className="space-y-4">
                <Select
                  label="Time Control"
                  options={timeControlOptions}
                  value={selectedTimeControl}
                  onChange={(e) => setSelectedTimeControl(e.target.value)}
                  fullWidth
                  disabled={isSearching}
                />
                
                <div className="flex items-center">
                  <input
                    id="rated"
                    type="checkbox"
                    checked={isRated}
                    onChange={() => setIsRated(!isRated)}
                    disabled={isSearching}
                    className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                  />
                  <label htmlFor="rated" className="ml-2 block text-sm text-gray-700">
                    Rated match
                  </label>
                </div>
              </div>
              
              {error && (
                <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-md">
                  {error}
                </div>
              )}
              
              <Button
                onClick={handleFindMatch}
                variant="primary"
                fullWidth
                loading={isSearching}
              >
                {isSearching ? 'Finding opponent...' : 'Find Match'}
              </Button>
              
              {isSearching && (
                <div className="text-center text-gray-500 text-sm mt-2">
                  Searching for an opponent with similar rating...
                </div>
              )}
            </div>
          </Card>
          
          {/* Leaderboard Preview */}
          <Card title="Leaderboard">
            <div className="text-center py-8">
              <p className="text-gray-500">Leaderboard coming soon!</p>
              <p className="text-gray-400 text-sm mt-2">
                Play rated games to appear on the leaderboard
              </p>
            </div>
          </Card>
        </div>
        
        {/* Game Rules */}
        <Card className="mt-8" title="How to Play">
          <div className="prose prose-blue">
            <p>
              P2P Chess connects you directly with other players through secure peer-to-peer connections.
              Your moves are validated both by your opponent's client and our server.
            </p>
            
            <h3>Getting Started</h3>
            <ol>
              <li>Select your preferred time control</li>
              <li>Choose between rated or casual play</li>
              <li>Click "Find Match" to be paired with an opponent</li>
              <li>Make your moves by dragging and dropping pieces</li>
              <li>The game ends when one player checkmates the other, or when time runs out</li>
            </ol>
            
            <p>
              All games follow standard international chess rules including en passant, castling, and promotion.
            </p>
          </div>
        </Card>
      </div>
    </div>
  );
};

export default LobbyPage;
