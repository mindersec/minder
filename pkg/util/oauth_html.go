package util

const (
	// SessionHTML is the page displayed upon success when using a web browser during an interactive OAuth token flow.
	SessionHTML = `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Mediator</title>
	<style>
		body {
			background-color: #ffffff;
			font-family: Arial, sans-serif;
			text-align: center;
			margin-top: 20vh;
		}

		h1 {
			font-weight: bold;
			font-size: 32px;
			color: #000000;
			margin-bottom: 20px;
		}
	</style>
</head>
<body>
	<h1>Mediator enrollment complete</h1>
	<p>You can now close this window and return to the CLI.</p>
</body>
</html>
`
)
