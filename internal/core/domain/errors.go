package domain

import "errors"

var (
	ErrPollNotFound  = errors.New("poll not found")
	ErrInvalidPollID = errors.New("invalid poll id")
	ErrInvalidOption = errors.New("invalid option for this poll")
	ErrAlreadyVoted  = errors.New("user has already voted")
	ErrDidNotVoted   = errors.New("user did not vote on this poll")
	ErrInternal      = errors.New("internal server error")
)
