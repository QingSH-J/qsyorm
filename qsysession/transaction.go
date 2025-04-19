package qsysession

// Begin starts a transaction
func (s *Session) Begin() (err error) {
	if s.tx != nil {
		return
	}

	s.Logger.Info("transaction begin")
	s.tx, err = s.db.Begin()
	if err != nil {
		s.Logger.Error("failed to begin transaction: %v", err)
		return
	}

	return
}

// Commit commits the transaction
func (s *Session) Commit() (err error) {
	if s.tx == nil {
		return
	}

	s.Logger.Info("transaction commit")
	err = s.tx.Commit()
	if err != nil {
		s.Logger.Error("failed to commit transaction: %v", err)
		return
	}

	s.tx = nil
	return
}

// Rollback aborts the transaction
func (s *Session) Rollback() (err error) {
	if s.tx == nil {
		return
	}

	s.Logger.Info("transaction rollback")
	err = s.tx.Rollback()
	if err != nil {
		s.Logger.Error("failed to rollback transaction: %v", err)
		return
	}

	s.tx = nil
	return
}

// Transaction executes a function within a transaction
// If the function returns an error, the transaction is rolled back
// Otherwise, the transaction is committed
func (s *Session) Transaction(f func(*Session) error) (err error) {
	if err = s.Begin(); err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = s.Rollback()
			panic(p) // re-throw panic after rollback
		} else if err != nil {
			_ = s.Rollback() // rollback on error
		} else {
			err = s.Commit() // commit when no error
		}
	}()

	err = f(s)
	return
}
