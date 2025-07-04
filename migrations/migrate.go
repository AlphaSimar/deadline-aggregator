func RunMigrations(db *sql.DB) error {
	schema, err := os.ReadFile("migrations/001_create_tables.sql")
	if err != nil {
		return err
	}
	_, err = db.Exec(string(schema))
	return err
}
