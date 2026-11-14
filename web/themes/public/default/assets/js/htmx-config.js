// HTMX treats 4xx and 5xx as failures and does not swap their responses by
// default, which would hide the error fragments the admin endpoints return. The
// handlers answer with the status that describes the problem, so the fragments
// are revealed here rather than by answering 200 for every failure.
//
// This file is loaded with defer directly after htmx.min.js, so the deferred
// scripts run in order and htmx is defined by the time this line runs.
htmx.config.responseHandling = [
	{code: "204", swap: false},
	{code: "[23]..", swap: true},
	{code: "[45]..", swap: true, error: true},
];
