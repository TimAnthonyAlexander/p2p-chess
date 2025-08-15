import type { MatchDetails, WebRTCMessage } from '../types';

class WebRTCService {
  private peerConnection: RTCPeerConnection | null = null;
  private dataChannel: RTCDataChannel | null = null;
  private websocket: WebSocket | null = null;
  private matchDetails: MatchDetails | null = null;
  private messageHandler: ((message: WebRTCMessage) => void) | null = null;
  private connectionStateHandler: ((state: RTCPeerConnectionState) => void) | null = null;
  
  constructor() {
    // Initialize empty
  }

  // Initialize WebRTC with match details and callback handlers
  initialize(
    matchDetails: MatchDetails,
    onMessage: (message: WebRTCMessage) => void,
    onConnectionStateChange: (state: RTCPeerConnectionState) => void
  ): void {
    this.matchDetails = matchDetails;
    this.messageHandler = onMessage;
    this.connectionStateHandler = onConnectionStateChange;
    
    // Set up the WebSocket signaling connection
    this.initWebSocket();
  }

  private initWebSocket(): void {
    if (!this.matchDetails?.joinToken) {
      console.error("No join token available for WebSocket connection");
      return;
    }
    
    const wsProtocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
    const wsUrl = `${wsProtocol}://localhost:8081/v1/ws/signal?token=${this.matchDetails.joinToken}`;
    
    this.websocket = new WebSocket(wsUrl);
    
    this.websocket.onopen = () => {
      console.log("WebSocket connection established");
      
      // Send join message to the signaling server
      if (this.websocket && this.matchDetails) {
        this.websocket.send(JSON.stringify({
          action: "join",
          matchId: this.matchDetails.matchId
        }));
        
        // Initialize the peer connection after joining
        this.initPeerConnection();
        
        // If we're the white player, we create the offer
        if (this.matchDetails.color === 'white') {
          setTimeout(() => this.createOffer(), 1000);
        }
      }
    };
    
    this.websocket.onmessage = (event) => {
      const message = JSON.parse(event.data);
      
      switch (message.action) {
        case 'offer':
          this.handleOfferMessage(message);
          break;
        case 'answer':
          this.handleAnswerMessage(message);
          break;
        case 'ice':
          this.handleIceCandidate(message);
          break;
        default:
          console.log('Unhandled signaling message:', message);
      }
    };
    
    this.websocket.onerror = (error) => {
      console.error("WebSocket error:", error);
    };
    
    this.websocket.onclose = () => {
      console.log("WebSocket connection closed");
    };
  }

  private initPeerConnection(): void {
    const iceServers = this.matchDetails?.iceServers || [
      { urls: 'stun:stun.l.google.com:19302' }
    ];
    
    // Add TURN server if provided in match details
    if (this.matchDetails?.turn) {
      iceServers.push({
        urls: this.matchDetails.turn.url,
        username: this.matchDetails.turn.username,
        credential: this.matchDetails.turn.password
      });
    }
    
    this.peerConnection = new RTCPeerConnection({ iceServers });
    
    // Set up event handlers
    this.peerConnection.onicecandidate = (event) => {
      if (event.candidate && this.websocket) {
        this.websocket.send(JSON.stringify({
          action: 'ice',
          candidate: event.candidate
        }));
      }
    };
    
    this.peerConnection.onconnectionstatechange = () => {
      if (this.peerConnection && this.connectionStateHandler) {
        this.connectionStateHandler(this.peerConnection.connectionState);
      }
    };
    
    // Create data channel if we're the initiator (white)
    if (this.matchDetails?.color === 'white') {
      this.setupDataChannel(this.peerConnection.createDataChannel('moves'));
    } else {
      // Otherwise, wait for the data channel from the other peer
      this.peerConnection.ondatachannel = (event) => {
        this.setupDataChannel(event.channel);
      };
    }
  }

  private setupDataChannel(channel: RTCDataChannel): void {
    this.dataChannel = channel;
    
    channel.onopen = () => {
      console.log('Data channel opened');
    };
    
    channel.onclose = () => {
      console.log('Data channel closed');
    };
    
    channel.onerror = (error) => {
      console.error('Data channel error:', error);
    };
    
    channel.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data) as WebRTCMessage;
        if (this.messageHandler) {
          this.messageHandler(message);
        }
      } catch (error) {
        console.error('Error parsing message:', error);
      }
    };
  }

  private async createOffer(): Promise<void> {
    if (!this.peerConnection || !this.websocket) return;
    
    try {
      const offer = await this.peerConnection.createOffer();
      await this.peerConnection.setLocalDescription(offer);
      
      this.websocket.send(JSON.stringify({
        action: 'offer',
        sdp: offer.sdp
      }));
    } catch (error) {
      console.error('Error creating offer:', error);
    }
  }

  private async handleOfferMessage(message: any): Promise<void> {
    if (!this.peerConnection || !this.websocket) return;
    
    try {
      await this.peerConnection.setRemoteDescription(
        new RTCSessionDescription({
          type: 'offer',
          sdp: message.sdp
        })
      );
      
      const answer = await this.peerConnection.createAnswer();
      await this.peerConnection.setLocalDescription(answer);
      
      this.websocket.send(JSON.stringify({
        action: 'answer',
        sdp: answer.sdp
      }));
    } catch (error) {
      console.error('Error handling offer:', error);
    }
  }

  private async handleAnswerMessage(message: any): Promise<void> {
    if (!this.peerConnection) return;
    
    try {
      await this.peerConnection.setRemoteDescription(
        new RTCSessionDescription({
          type: 'answer',
          sdp: message.sdp
        })
      );
    } catch (error) {
      console.error('Error handling answer:', error);
    }
  }

  private async handleIceCandidate(message: any): Promise<void> {
    if (!this.peerConnection) return;
    
    try {
      await this.peerConnection.addIceCandidate(message.candidate);
    } catch (error) {
      console.error('Error adding ICE candidate:', error);
    }
  }

  // Send a message to the peer
  sendMessage(message: WebRTCMessage): boolean {
    if (!this.dataChannel || this.dataChannel.readyState !== 'open') {
      console.error('Cannot send message, data channel not open');
      return false;
    }
    
    try {
      this.dataChannel.send(JSON.stringify(message));
      return true;
    } catch (error) {
      console.error('Error sending message:', error);
      return false;
    }
  }

  // Close the connection
  close(): void {
    if (this.dataChannel) {
      this.dataChannel.close();
      this.dataChannel = null;
    }
    
    if (this.peerConnection) {
      this.peerConnection.close();
      this.peerConnection = null;
    }
    
    if (this.websocket) {
      this.websocket.close();
      this.websocket = null;
    }
    
    this.messageHandler = null;
    this.connectionStateHandler = null;
  }
}

// Export as a singleton
export default new WebRTCService();
