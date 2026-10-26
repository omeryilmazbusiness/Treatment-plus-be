// Package outbox is the transactional outbox port.
// HTTP handlers never wait for PDF, AI, mail, or expiry jobs.
// The worker process drains this queue. Phase 0 has no jobs.
package outbox
