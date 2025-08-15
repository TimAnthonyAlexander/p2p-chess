# P2P Chess Frontend

A React-based frontend for P2P Chess, enabling peer-to-peer chess gameplay through WebRTC.

## Features

- User authentication (login/registration)
- Matchmaking for finding opponents
- Real-time chess gameplay via peer-to-peer WebRTC
- Secure move validation
- Interactive chess board with drag-and-drop functionality
- Time controls for different game formats
- Game history and move notation

## Tech Stack

- React 19 with TypeScript
- Vite for fast development and building
- React Router for client-side routing
- TailwindCSS for styling
- chess.js for chess logic
- react-chessboard for the chess UI
- WebRTC for peer-to-peer communication

## Project Structure

```
/frontend
├── /public         # Static assets
├── /src
│   ├── /assets     # Images, fonts, etc.
│   ├── /components # Reusable UI components
│   ├── /context    # React Context providers
│   ├── /pages      # Page components
│   ├── /services   # API and WebRTC services
│   ├── /types      # TypeScript type definitions
│   ├── /utils      # Utility functions
│   ├── App.tsx     # Main app component with routing
│   ├── index.css   # Global styles
│   └── main.tsx    # Entry point
├── .eslintrc.js    # ESLint configuration
├── index.html      # HTML template
├── package.json    # Dependencies and scripts
├── tsconfig.json   # TypeScript configuration
└── vite.config.ts  # Vite configuration
```

## Getting Started

### Prerequisites

- Node.js 16+ and npm

### Installation

1. Clone the repository
2. Navigate to the frontend directory:
   ```bash
   cd p2p-chess/frontend
   ```
3. Install dependencies:
   ```bash
   npm install
   ```

### Development

Start the development server:
```bash
npm run dev
```

This will start the Vite development server, typically at http://localhost:5173.

### Building for Production

```bash
npm run build
```

The built files will be in the `dist` directory.

### Backend Connection

The frontend is configured to connect to a backend server running on `http://localhost:8081` by default. You can change this in `src/services/api.ts` by updating the `API_BASE_URL` constant.

## Authentication

User authentication uses JWT tokens stored in localStorage. After login, the token is automatically included in API requests through the `Authorization` header.

## Gameplay Flow

1. User logs in or registers
2. User visits the lobby and selects game preferences (time control, rated/casual)
3. User clicks "Find Match" to enter the matchmaking queue
4. When paired with an opponent, the user is redirected to the game page
5. WebRTC connection is established for peer-to-peer gameplay
6. Users can make moves, resign, or claim victory on time
7. Game results are recorded on the server