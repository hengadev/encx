// Package industry demonstrates HIPAA-compliant healthcare data encryption
//
// Context7 Tags: healthcare, HIPAA-compliance, medical-records, PHI-protection, patient-data
// Complexity: Industry-Specific
// Use Case: Protecting patient health information according to HIPAA requirements

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hengadev/encx"
)

// Patient represents a patient record with HIPAA-compliant encryption
// All PHI (Protected Health Information) fields are encrypted
type Patient struct {
	// Public identifiers (not PHI under HIPAA)
	ID           int       `json:"id" db:"id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
	DepartmentID int       `json:"department_id" db:"department_id"`

	// PHI - Personal Identifiers (encrypted + searchable)
	MRN              string `encx:"encrypt,hash_basic" json:"mrn" db:"mrn"`                     // Medical Record Number
	MRNEncrypted     []byte `json:"-" db:"mrn_encrypted"`
	MRNHash          string `json:"-" db:"mrn_hash"`

	Email            string `encx:"encrypt,hash_basic" json:"email" db:"email"`
	EmailEncrypted   []byte `json:"-" db:"email_encrypted"`
	EmailHash        string `json:"-" db:"email_hash"`

	Phone            string `encx:"encrypt,hash_basic" json:"phone" db:"phone"`
	PhoneEncrypted   []byte `json:"-" db:"phone_encrypted"`
	PhoneHash        string `json:"-" db:"phone_hash"`

	// PHI - Personal Information (encrypted only)
	FirstName        string `encx:"encrypt" json:"first_name" db:"first_name"`
	FirstNameEncrypted []byte `json:"-" db:"first_name_encrypted"`

	LastName         string `encx:"encrypt" json:"last_name" db:"last_name"`
	LastNameEncrypted []byte `json:"-" db:"last_name_encrypted"`

	DateOfBirth      string `encx:"encrypt" json:"date_of_birth" db:"date_of_birth"`
	DateOfBirthEncrypted []byte `json:"-" db:"date_of_birth_encrypted"`

	// PHI - Sensitive Identifiers (hashed only - no decryption)
	SSN              string `encx:"hash_secure" json:"-" db:"ssn"`
	SSNHashSecure    string `json:"-" db:"ssn_hash_secure"`

	InsuranceID      string `encx:"hash_secure" json:"-" db:"insurance_id"`
	InsuranceIDHashSecure string `json:"-" db:"insurance_id_hash_secure"`

	// PHI - Address Information (encrypted)
	Address          string `encx:"encrypt" json:"address" db:"address"`
	AddressEncrypted []byte `json:"-" db:"address_encrypted"`

	City             string `encx:"encrypt" json:"city" db:"city"`
	CityEncrypted    []byte `json:"-" db:"city_encrypted"`

	State            string `encx:"encrypt" json:"state" db:"state"`
	StateEncrypted   []byte `json:"-" db:"state_encrypted"`

	ZipCode          string `encx:"encrypt" json:"zip_code" db:"zip_code"`
	ZipCodeEncrypted []byte `json:"-" db:"zip_code_encrypted"`

	// Required encryption fields
	DEK              []byte `json:"-" db:"dek"`
	DEKEncrypted     []byte `json:"-" db:"dek_encrypted"`
	KeyVersion       int    `json:"-" db:"key_version"`
}

// MedicalRecord represents a medical record with encrypted health information
type MedicalRecord struct {
	// Basic information
	ID             int       `json:"id" db:"id"`
	PatientID      int       `json:"patient_id" db:"patient_id"`
	ProviderID     int       `json:"provider_id" db:"provider_id"`
	RecordDate     time.Time `json:"record_date" db:"record_date"`
	RecordType     string    `json:"record_type" db:"record_type"` // e.g., "visit", "lab", "imaging"

	// PHI - Medical Information (encrypted)
	ChiefComplaint   string `encx:"encrypt" json:"chief_complaint" db:"chief_complaint"`
	ChiefComplaintEncrypted []byte `json:"-" db:"chief_complaint_encrypted"`

	Diagnosis        string `encx:"encrypt" json:"diagnosis" db:"diagnosis"`
	DiagnosisEncrypted []byte `json:"-" db:"diagnosis_encrypted"`

	Treatment        string `encx:"encrypt" json:"treatment" db:"treatment"`
	TreatmentEncrypted []byte `json:"-" db:"treatment_encrypted"`

	Medications      string `encx:"encrypt" json:"medications" db:"medications"`
	MedicationsEncrypted []byte `json:"-" db:"medications_encrypted"`

	Notes            string `encx:"encrypt" json:"notes" db:"notes"`
	NotesEncrypted   []byte `json:"-" db:"notes_encrypted"`

	// Searchable medical codes (hashed for research/reporting)
	ICD10Code        string `encx:"hash_basic" json:"icd10_code" db:"icd10_code"`
	ICD10CodeHash    string `json:"-" db:"icd10_code_hash"`

	CPTCode          string `encx:"hash_basic" json:"cpt_code" db:"cpt_code"`
	CPTCodeHash      string `json:"-" db:"cpt_code_hash"`

	// Required encryption fields
	DEK              []byte `json:"-" db:"dek"`
	DEKEncrypted     []byte `json:"-" db:"dek_encrypted"`
	KeyVersion       int    `json:"-" db:"key_version"`
}

// PatientProfile represents a safe view of patient data for display
// This excludes sensitive identifiers while providing necessary medical context
type PatientProfile struct {
	ID          int    `json:"id"`
	MRN         string `json:"mrn"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	DateOfBirth string `json:"date_of_birth"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Address     string `json:"address"`
	City        string `json:"city"`
	State       string `json:"state"`
	ZipCode     string `json:"zip_code"`
	// Note: SSN and Insurance ID intentionally excluded
}

func main() {
	ctx := context.Background()

	// Use production-grade crypto setup for healthcare
	crypto, err := encx.NewTestCrypto(nil) // In production, use proper KMS
	if err != nil {
		log.Fatal("Failed to create crypto service:", err)
	}

	// Example 1: Patient registration with PHI protection
	fmt.Println("=== HIPAA-Compliant Patient Registration ===")

	patient := &Patient{
		ID:            1001,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		DepartmentID:  5, // Cardiology
		MRN:           "MRN-2024-001001",
		Email:         "jane.doe@email.com",
		Phone:         "+1-555-987-6543",
		FirstName:     "Jane",
		LastName:      "Doe",
		DateOfBirth:   "1985-07-15",
		SSN:           "987-65-4321",
		InsuranceID:   "INS-ABC-123456",
		Address:       "456 Oak Avenue",
		City:          "Boston",
		State:         "MA",
		ZipCode:       "02101",
	}

	fmt.Printf("Original patient data contains PHI:\n")
	fmt.Printf("  Name: %s %s\n", patient.FirstName, patient.LastName)
	fmt.Printf("  MRN: %s\n", patient.MRN)
	fmt.Printf("  SSN: %s\n", patient.SSN)
	fmt.Printf("  DOB: %s\n", patient.DateOfBirth)

	// Encrypt all PHI according to HIPAA requirements
	if err := crypto.ProcessStruct(ctx, patient); err != nil {
		log.Fatal("Failed to encrypt patient PHI:", err)
	}

	fmt.Printf("\nAfter HIPAA-compliant encryption:\n")
	fmt.Printf("  FirstName: '%s' (cleared)\n", patient.FirstName)
	fmt.Printf("  MRN: '%s' (cleared)\n", patient.MRN)
	fmt.Printf("  SSN: '%s' (cleared)\n", patient.SSN)
	fmt.Printf("  MRN searchable hash: %s...\n", patient.MRNHash[:16])
	fmt.Printf("  Email searchable hash: %s...\n", patient.EmailHash[:16])
	fmt.Printf("  SSN secure hash: %s...\n", patient.SSNHashSecure[:20])

	// Example 2: Medical record with encrypted health information
	fmt.Println("\n=== Medical Record Encryption ===")

	record := &MedicalRecord{
		ID:              2001,
		PatientID:       patient.ID,
		ProviderID:      301,
		RecordDate:      time.Now(),
		RecordType:      "visit",
		ChiefComplaint:  "Chest pain and shortness of breath",
		Diagnosis:       "Acute coronary syndrome, rule out myocardial infarction",
		Treatment:       "ECG, troponin levels, cardiac catheterization recommended",
		Medications:     "Aspirin 325mg, Metoprolol 25mg BID, Atorvastatin 40mg",
		Notes:           "Patient presents with 2-hour history of substernal chest pain...",
		ICD10Code:       "I20.9", // Angina pectoris, unspecified
		CPTCode:         "99214", // Office visit, moderate complexity
	}

	fmt.Printf("Original medical record contains sensitive health data:\n")
	fmt.Printf("  Chief Complaint: %s\n", record.ChiefComplaint)
	fmt.Printf("  Diagnosis: %s\n", record.Diagnosis)
	fmt.Printf("  Medications: %s\n", record.Medications)

	// Encrypt medical information
	if err := crypto.ProcessStruct(ctx, record); err != nil {
		log.Fatal("Failed to encrypt medical record:", err)
	}

	fmt.Printf("\nAfter encryption:\n")
	fmt.Printf("  Chief Complaint: '%s' (cleared)\n", record.ChiefComplaint)
	fmt.Printf("  Diagnosis: '%s' (cleared)\n", record.Diagnosis)
	fmt.Printf("  Medications: '%s' (cleared)\n", record.Medications)
	fmt.Printf("  ICD10 hash: %s...\n", record.ICD10CodeHash[:16])
	fmt.Printf("  CPT hash: %s...\n", record.CPTCodeHash[:16])

	// Example 3: HIPAA-compliant data access patterns
	fmt.Println("\n=== HIPAA-Compliant Data Access ===")

	// Patient lookup by MRN (authorized access)
	searchPatient := &Patient{MRN: "MRN-2024-001001"}
	if err := crypto.ProcessStruct(ctx, searchPatient); err != nil {
		log.Fatal("Failed to hash MRN for search:", err)
	}

	fmt.Printf("Patient lookup by MRN:\n")
	fmt.Printf("  Search hash: %s...\n", searchPatient.MRNHash[:16])
	fmt.Printf("  Matches patient: %t\n", searchPatient.MRNHash == patient.MRNHash)

	// Create safe patient profile (decrypt only necessary data)
	if err := crypto.DecryptStruct(ctx, patient); err != nil {
		log.Fatal("Failed to decrypt patient data:", err)
	}

	profile := PatientProfile{
		ID:          patient.ID,
		MRN:         patient.MRN,
		FirstName:   patient.FirstName,
		LastName:    patient.LastName,
		DateOfBirth: patient.DateOfBirth,
		Email:       patient.Email,
		Phone:       patient.Phone,
		Address:     patient.Address,
		City:        patient.City,
		State:       patient.State,
		ZipCode:     patient.ZipCode,
		// SSN and Insurance ID intentionally excluded
	}

	fmt.Printf("\nSafe patient profile (excludes sensitive identifiers):\n")
	fmt.Printf("  %+v\n", profile)
}

// HIPAA-compliant service patterns

// PatientService implements HIPAA-compliant patient data handling
type PatientService struct {
	crypto *encx.Crypto
}

func NewPatientService(crypto *encx.Crypto) *PatientService {
	return &PatientService{crypto: crypto}
}

// RegisterPatient implements secure patient registration
func (s *PatientService) RegisterPatient(ctx context.Context, patient *Patient) error {
	// Validate required fields
	if patient.FirstName == "" || patient.LastName == "" || patient.DateOfBirth == "" {
		return fmt.Errorf("first name, last name, and date of birth are required")
	}

	// Encrypt all PHI
	if err := s.crypto.ProcessStruct(ctx, patient); err != nil {
		return fmt.Errorf("failed to encrypt patient PHI: %w", err)
	}

	// In production: save to HIPAA-compliant database
	// - Use encrypted database connections
	// - Log all access attempts
	// - Implement proper backup encryption
	fmt.Printf("Patient registered with encrypted PHI\n")

	return nil
}

// FindPatientByMRN implements secure patient lookup
func (s *PatientService) FindPatientByMRN(ctx context.Context, mrn string) (*Patient, error) {
	// Generate search hash
	searchPatient := &Patient{MRN: mrn}
	if err := s.crypto.ProcessStruct(ctx, searchPatient); err != nil {
		return nil, fmt.Errorf("failed to hash MRN: %w", err)
	}

	// In production: query database by hash
	// SELECT * FROM patients WHERE mrn_hash = ?
	fmt.Printf("Database query: patients.mrn_hash = %s...\n", searchPatient.MRNHash[:16])

	// Mock result - in production, load from database and decrypt
	return &Patient{MRNHash: searchPatient.MRNHash}, nil
}

// GetPatientProfile returns safe patient data for authorized access
func (s *PatientService) GetPatientProfile(ctx context.Context, patientID int, requesterRole string) (*PatientProfile, error) {
	// Implement role-based access control
	if !s.isAuthorizedForPatientData(requesterRole) {
		return nil, fmt.Errorf("insufficient privileges to access patient data")
	}

	// In production: load encrypted patient from database
	// patient := loadEncryptedPatient(patientID)

	// Decrypt only for authorized access
	// err := s.crypto.DecryptStruct(ctx, patient)

	// Log access for HIPAA audit trail
	s.logPatientAccess(ctx, patientID, requesterRole, "profile_access")

	// Return safe profile (excludes SSN, Insurance ID)
	return &PatientProfile{
		ID:        patientID,
		FirstName: "Jane", // In production, from decrypted data
		LastName:  "Doe",
		// ... other safe fields
	}, nil
}

// Medical record service
type MedicalRecordService struct {
	crypto *encx.Crypto
}

// CreateMedicalRecord securely stores medical information
func (s *MedicalRecordService) CreateMedicalRecord(ctx context.Context, record *MedicalRecord) error {
	// Validate medical record
	if record.PatientID == 0 || record.ProviderID == 0 {
		return fmt.Errorf("patient ID and provider ID are required")
	}

	// Encrypt medical information
	if err := s.crypto.ProcessStruct(ctx, record); err != nil {
		return fmt.Errorf("failed to encrypt medical record: %w", err)
	}

	// Log creation for audit trail
	s.logMedicalRecordAccess(ctx, record.PatientID, record.ProviderID, "create")

	return nil
}

// FindRecordsByDiagnosisCode enables medical research while protecting PHI
func (s *MedicalRecordService) FindRecordsByDiagnosisCode(ctx context.Context, icd10Code string) ([]int, error) {
	// Generate search hash for ICD-10 code
	searchRecord := &MedicalRecord{ICD10Code: icd10Code}
	if err := s.crypto.ProcessStruct(ctx, searchRecord); err != nil {
		return nil, fmt.Errorf("failed to hash ICD-10 code: %w", err)
	}

	// Query by hash enables research without exposing PHI
	// SELECT patient_id FROM medical_records WHERE icd10_code_hash = ?
	fmt.Printf("Research query: medical_records.icd10_code_hash = %s...\n",
		searchRecord.ICD10CodeHash[:16])

	// Return de-identified results
	return []int{1001, 1002, 1003}, nil
}

// HIPAA compliance helper functions

func (s *PatientService) isAuthorizedForPatientData(role string) bool {
	authorizedRoles := []string{"doctor", "nurse", "admin", "patient"}
	for _, authorized := range authorizedRoles {
		if role == authorized {
			return true
		}
	}
	return false
}

func (s *PatientService) logPatientAccess(ctx context.Context, patientID int, role, action string) {
	// HIPAA requires detailed audit logs
	fmt.Printf("AUDIT LOG: User with role '%s' performed '%s' on patient %d at %v\n",
		role, action, patientID, time.Now())
}

func (s *MedicalRecordService) logMedicalRecordAccess(ctx context.Context, patientID, providerID int, action string) {
	fmt.Printf("AUDIT LOG: Provider %d performed '%s' on patient %d medical record at %v\n",
		providerID, action, patientID, time.Now())
}

/*
HIPAA Compliance Features Demonstrated:

1. **PHI Encryption**: All Protected Health Information encrypted at rest
2. **Searchable Identifiers**: Hash MRN, email, phone for authorized lookups
3. **Secure Identifiers**: SSN, Insurance ID hashed only (no decryption)
4. **Role-Based Access**: Different data access based on user role
5. **Audit Logging**: All access attempts logged for compliance
6. **Data Minimization**: Return only necessary data for each use case
7. **De-identification**: Research queries without exposing patient identity

Database Schema for HIPAA Compliance:

CREATE TABLE patients (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE,
    department_id INTEGER,

    -- Searchable encrypted identifiers
    mrn_encrypted BYTEA NOT NULL,
    mrn_hash VARCHAR(64) UNIQUE NOT NULL,
    email_encrypted BYTEA,
    email_hash VARCHAR(64),
    phone_encrypted BYTEA,
    phone_hash VARCHAR(64),

    -- Personal information (encrypted)
    first_name_encrypted BYTEA NOT NULL,
    last_name_encrypted BYTEA NOT NULL,
    date_of_birth_encrypted BYTEA NOT NULL,
    address_encrypted BYTEA,
    city_encrypted BYTEA,
    state_encrypted BYTEA,
    zip_code_encrypted BYTEA,

    -- Sensitive identifiers (hashed only)
    ssn_hash_secure TEXT NOT NULL,
    insurance_id_hash_secure TEXT,

    -- Encryption metadata
    dek_encrypted BYTEA NOT NULL,
    key_version INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Audit table for HIPAA compliance
CREATE TABLE patient_access_log (
    id SERIAL PRIMARY KEY,
    patient_id INTEGER REFERENCES patients(id),
    user_id INTEGER,
    user_role VARCHAR(50),
    action VARCHAR(100),
    access_time TIMESTAMP DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT
);

HIPAA Security Requirements Met:
✅ Encryption at rest (AES-GCM)
✅ Access controls (role-based authorization)
✅ Audit trails (comprehensive logging)
✅ Data minimization (safe profiles)
✅ Secure key management (DEK/KEK architecture)
✅ De-identification for research
✅ Secure transmission (use HTTPS in production)

Production Considerations:
- Use proper HSM/KMS for key management (not NewTestCrypto)
- Implement proper user authentication and authorization
- Use encrypted database connections (TLS)
- Regular security audits and penetration testing
- Backup encryption and secure key escrow
- Employee training on HIPAA compliance
- Business Associate Agreements (BAAs) with vendors
*/