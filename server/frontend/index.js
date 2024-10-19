const socket = new WebSocket("ws://localhost:8080/ws");

socket.onopen = function(event) {
	console.log("Connected to server!");

	// Send player position every second (simulate movement)
	setInterval(() => {
		const pos = [Math.random() * 100, Math.random() * 100];
		socket.send(JSON.stringify(pos));
		console.log("Sent position: ", pos);
	}, 5000);
};

socket.onmessage = function(event) {
	const playersData = JSON.parse(event.data);
	console.log("Received players data:", playersData);
};

socket.onclose = function(event) {
	console.log("Disconnected from server.");
};
