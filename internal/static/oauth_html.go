// Copyright 2023 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package static

const (
	// InteractiveSuccessHTML is the page displayed upon success when using a web browser during an interactive Oauth token flow.
	InteractiveSuccessHTML = `<html>
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
