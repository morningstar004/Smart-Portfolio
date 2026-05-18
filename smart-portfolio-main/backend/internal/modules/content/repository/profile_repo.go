package repository

import (
	"context"

	"github.com/morningstar004/smart-portfolio/internal/database"
	"github.com/morningstar004/smart-portfolio/internal/modules/content/model"
)

type ProfileRepository struct {
	pg *database.Postgres
}

func NewProfileRepository(pg *database.Postgres) *ProfileRepository {
	return &ProfileRepository{pg: pg}
}

func (r *ProfileRepository) GetProfile(ctx context.Context) (*model.Profile, error) {
	query := `SELECT id, first_name, last_name, primary_role, specialization, location, summary, updated_at FROM profile LIMIT 1`
	row := r.pg.Pool.QueryRow(ctx, query)
	var p model.Profile
	err := row.Scan(&p.ID, &p.FirstName, &p.LastName, &p.PrimaryRole, &p.Specialization, &p.Location, &p.Summary, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProfileRepository) GetEducation(ctx context.Context) ([]model.Education, error) {
	query := `SELECT id, institution, degree, location, start_date, end_date, gpa, coursework, created_at FROM education ORDER BY start_date DESC`
	rows, err := r.pg.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.Education
	for rows.Next() {
		var e model.Education
		err := rows.Scan(&e.ID, &e.Institution, &e.Degree, &e.Location, &e.StartDate, &e.EndDate, &e.GPA, &e.Coursework, &e.CreatedAt)
		if err != nil {
			return nil, err
		}
		results = append(results, e)
	}
	return results, nil
}

func (r *ProfileRepository) GetExperience(ctx context.Context) ([]model.Experience, error) {
	query := `SELECT id, company, role, location, start_date, end_date, summary, tech_stack, created_at FROM experience ORDER BY start_date DESC`
	rows, err := r.pg.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.Experience
	for rows.Next() {
		var e model.Experience
		err := rows.Scan(&e.ID, &e.Company, &e.Role, &e.Location, &e.StartDate, &e.EndDate, &e.Summary, &e.TechStack, &e.CreatedAt)
		if err != nil {
			return nil, err
		}
		results = append(results, e)
	}
	return results, nil
}

func (r *ProfileRepository) GetCertifications(ctx context.Context) ([]model.Certification, error) {
	query := `SELECT id, name, issuer, issue_date, url, created_at FROM certifications ORDER BY issue_date DESC`
	rows, err := r.pg.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.Certification
	for rows.Next() {
		var c model.Certification
		err := rows.Scan(&c.ID, &c.Name, &c.Issuer, &c.IssueDate, &c.URL, &c.CreatedAt)
		if err != nil {
			return nil, err
		}
		results = append(results, c)
	}
	return results, nil
}

func (r *ProfileRepository) GetAchievements(ctx context.Context) ([]model.Achievement, error) {
	query := `SELECT id, title, metric, description, date, created_at FROM achievements ORDER BY date DESC`
	rows, err := r.pg.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.Achievement
	for rows.Next() {
		var a model.Achievement
		err := rows.Scan(&a.ID, &a.Title, &a.Metric, &a.Description, &a.Date, &a.CreatedAt)
		if err != nil {
			return nil, err
		}
		results = append(results, a)
	}
	return results, nil
}

func (r *ProfileRepository) GetSkills(ctx context.Context) ([]model.Skill, error) {
	query := `SELECT id, category, name, created_at FROM skills ORDER BY category, name`
	rows, err := r.pg.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.Skill
	for rows.Next() {
		var s model.Skill
		err := rows.Scan(&s.ID, &s.Category, &s.Name, &s.CreatedAt)
		if err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, nil
}
