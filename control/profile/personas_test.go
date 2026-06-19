package profile

import (
	"testing"
)

func TestVocab_DefaultSetRecognized(t *testing.T) {
	for _, p := range []string{PersonaGeneral, PersonaICU, PersonaNursing, PersonaRadiology, PersonaEmergency} {
		if !ValidPersona(p) {
			t.Errorf("default persona %q not recognized", p)
		}
	}
	for _, d := range []string{DomainGeneral, DomainHealthcare, DomainFinance, DomainLegal} {
		if !ValidDomain(d) {
			t.Errorf("default domain %q not recognized", d)
		}
	}
	if !ValidPersona("") || !ValidDomain("") {
		t.Error("blank persona/domain must be valid")
	}
}

func TestVocab_RegisterExtends(t *testing.T) {
	token := "test-persona-unique"
	defer func() { _ = DeletePersona(token) }()
	if ValidPersona(token) {
		t.Fatal("precondition: token should not exist")
	}
	RegisterPersona(token)
	if !ValidPersona(token) {
		t.Errorf("RegisterPersona did not register %q", token)
	}
	found := false
	for _, e := range ListPersonas() {
		if e.Token == token {
			found = true
		}
	}
	if !found {
		t.Errorf("ListPersonas did not include registered %q", token)
	}
}

func TestVocab_DeleteGeneralProtected(t *testing.T) {
	if err := DeletePersona(PersonaGeneral); err != ErrProtectedVocabulary {
		t.Errorf("deleting general persona: got %v, want ErrProtectedVocabulary", err)
	}
	if err := DeleteDomain(DomainGeneral); err != ErrProtectedVocabulary {
		t.Errorf("deleting general domain: got %v, want ErrProtectedVocabulary", err)
	}
}

func TestVocab_DeleteCustomAndBuiltinNonGeneral(t *testing.T) {
	// custom token: register then delete
	RegisterPersona("cardiology")
	if err := DeletePersona("cardiology"); err != nil {
		t.Errorf("deleting custom persona cardiology: %v", err)
	}
	if ValidPersona("cardiology") {
		t.Error("cardiology should be gone after delete")
	}
	// built-in non-general token (icu) is deletable too (only general is protected)
	if err := DeletePersona(PersonaICU); err != nil {
		t.Errorf("deleting built-in icu should succeed: %v", err)
	}
	RegisterPersona(PersonaICU) // restore for other tests
	// unknown token
	if err := DeletePersona("nope-not-real"); err != ErrUnknownVocabulary {
		t.Errorf("deleting unknown token: got %v, want ErrUnknownVocabulary", err)
	}
}

func TestVocab_ListMarksDeletable(t *testing.T) {
	RegisterPersona("temp-deletable")
	defer func() { _ = DeletePersona("temp-deletable") }()
	for _, e := range ListPersonas() {
		if e.Token == PersonaGeneral && e.Deletable {
			t.Error("general persona must be non-deletable")
		}
		if e.Token == "temp-deletable" && !e.Deletable {
			t.Error("custom persona must be deletable")
		}
	}
}
