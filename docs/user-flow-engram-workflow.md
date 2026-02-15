# Engram UI User Flow (4 Engrams, New Session Every Save)

This runbook demonstrates the full memory-continuity loop with one strict rule:

`Save Engram -> Continue in New Chat -> Pin all Engrams -> Continue conversation`

The flow uses the same default model for all sessions and captures screenshots only after assistant responses are rendered.

## Preconditions

1. API and web are running locally.
2. Open `http://localhost:5174/`.
3. Login credentials: `admin` / `admin123`.
4. Start from a clean stack/database.

## Scenario

Project: Orion Checkout incident continuity.

Goal: build a complete incident memory chain with four engrams across four linked sessions.

## Step-by-Step Screenshot Flow

### 1. Login page
![01](screenshots/user-flow/01-login.png)

### 2. Credentials filled
![02](screenshots/user-flow/02-login-filled.png)

### 3. Dashboard loaded
![03](screenshots/user-flow/03-dashboard.png)

### 4. Session 1 configured
![04](screenshots/user-flow/04-session-1-configured.png)

### 5. Session 1 created
![05](screenshots/user-flow/05-session-1-created.png)

### 6. Session 1 prompt ready
![06](screenshots/user-flow/06-session-1-prompt-ready.png)

### 7. Session 1 response rendered
![07](screenshots/user-flow/07-session-1-response-rendered.png)

### 8. Engram 1 save modal
![08](screenshots/user-flow/08-engram-1-save-modal.png)

### 9. Engram 1 saved
![09](screenshots/user-flow/09-engram-1-saved.png)

### 10. Session 2 (continued) created
![10](screenshots/user-flow/10-session-2-continued.png)

### 11. Session 2 pinned with Engram 1
![11](screenshots/user-flow/11-session-2-pin-engram-1.png)

### 12. Session 2 prompt ready
![12](screenshots/user-flow/12-session-2-prompt-ready.png)

### 13. Session 2 response rendered
![13](screenshots/user-flow/13-session-2-response-rendered.png)

### 14. Engram 2 save modal
![14](screenshots/user-flow/14-engram-2-save-modal.png)

### 15. Engram 2 saved
![15](screenshots/user-flow/15-engram-2-saved.png)

### 16. Session 3 (continued) created
![16](screenshots/user-flow/16-session-3-continued.png)

### 17. Session 3 pinned with all available engrams
![17](screenshots/user-flow/17-session-3-pin-engrams-1-2.png)

### 18. Session 3 prompt ready
![18](screenshots/user-flow/18-session-3-prompt-ready.png)

### 19. Session 3 response rendered
![19](screenshots/user-flow/19-session-3-response-rendered.png)

### 20. Engram 3 save modal
![20](screenshots/user-flow/20-engram-3-save-modal.png)

### 21. Engram 3 saved
![21](screenshots/user-flow/21-engram-3-saved.png)

### 22. Session 4 (continued) created
![22](screenshots/user-flow/22-session-4-continued.png)

### 23. Session 4 pinned with all available engrams
![23](screenshots/user-flow/23-session-4-pin-engrams-1-3.png)

### 24. Session 4 prompt ready
![24](screenshots/user-flow/24-session-4-prompt-ready.png)

### 25. Session 4 response rendered
![25](screenshots/user-flow/25-session-4-response-rendered.png)

### 26. Engram 4 save modal
![26](screenshots/user-flow/26-engram-4-save-modal.png)

### 27. Engram 4 saved and catalog visible
![27](screenshots/user-flow/27-engram-4-saved-final-catalog.png)

## Verification Checklist

1. Each engram is saved in its session before creating the next session.
2. Each new continued session pins all available engrams.
3. Final session can reference a multi-engram memory chain.
4. Search panel shows the engram catalog for cross-session reuse.
