document.addEventListener('DOMContentLoaded', () => {

     // --- Firebase Configuration ---
     const firebaseConfig = {
          apiKey: "AIzaSyBSzixG2InF6Y5dEamzGX0Ka1XQWfnd-uA",
          authDomain: "m-b-s-e-moonpoly-v2-0-0-zutmic.firebaseapp.com",
          projectId: "m-b-s-e-moonpoly-v2-0-0-zutmic",
          storageBucket: "m-b-s-e-moonpoly-v2-0-0-zutmic.firebasestorage.app",
          messagingSenderId: "652253468774",
          appId: "1:652253468774:web:d55fd115f85bc3d44a75a7"
     };
 
     // --- Initialize Firebase ---
     try {
         firebase.initializeApp(firebaseConfig);
         console.log("Firebase Initialized");
     } catch (error) {
         console.error("Error initializing Firebase:", error);
         displayMessage("Error initializing application. Please contact support.", true);
         return; // Stop execution if Firebase can't initialize
     }
 
 
     // --- DOM Element References ---
     const loginForm = document.getElementById('login-form');
     const emailInput = document.getElementById('email');
     const passwordInput = document.getElementById('password');
     const loginButton = document.getElementById('login-button');
     const messageArea = document.getElementById('message-area');
 
     // --- Event Listener for Form Submission ---
     loginForm.addEventListener('submit', (event) => {
         event.preventDefault(); // Prevent default HTML form submission behavior
 
         const email = emailInput.value.trim();
         const password = passwordInput.value; // No trim for password
 
         // Basic validation
         if (!email || !password) {
             displayMessage("Please enter both email and password.", true);
             return;
         }
 
         // Disable button and show feedback
         setLoadingState(true, "Logging in...");
 
         // --- Firebase Email/Password Sign-in ---
         firebase.auth().signInWithEmailAndPassword(email, password)
             .then((userCredential) => {
                 // Signed in successfully
                 const user = userCredential.user;
                 console.log("Sign-in successful for user:", user.uid);
                 displayMessage("Login successful. Preparing session...", false);
 
                 // Get the ID token (force refresh recommended for backend verification)
                 return user.getIdToken(/* forceRefresh */ true);
             })
             .then((idToken) => {
                 // --- Send Token to Go Application ---
                 console.log("Obtained ID Token.");
                 if (window.goSendToken && typeof window.goSendToken === 'function') {
                     try {
                          window.goSendToken(idToken); // Call the function exposed by Go/webview
                          console.log("Token sent to Go application.");
                          // Optional: Close window or show final message handled by Go after token receipt
                          // displayMessage("Session ready. You can close this window.", false);
                          // Note: The Go app might terminate the webview upon receiving the token.
                     } catch (bindError) {
                          console.error("Error calling Go binding function 'goSendToken':", bindError);
                          displayMessage("Login complete but failed to pass session to application.", true);
                          setLoadingState(false); // Re-enable button if Go binding failed
                     }
 
                 } else {
                     console.error("Go binding function 'goSendToken' is not defined or not a function.");
                     displayMessage("Login complete but cannot communicate with main application.", true);
                     setLoadingState(false); // Re-enable button if Go binding is missing
                 }
             })
             .catch((error) => {
                 // Handle Errors here.
                 console.error("Firebase Login Error:", error.code, error.message);
                 let userMessage = "Login failed. Please try again."; // Default message
 
                 // Provide more specific user feedback if possible
                 switch (error.code) {
                     case 'auth/invalid-email':
                         userMessage = "Invalid email address format.";
                         break;
                     case 'auth/user-disabled':
                         userMessage = "This account has been disabled.";
                         break;
                     case 'auth/user-not-found':
                     case 'auth/wrong-password':
                     case 'auth/invalid-credential': // Common code for wrong email/password
                         userMessage = "Invalid email or password.";
                         break;
                     // Add other specific error codes if needed
                 }
                 displayMessage(userMessage, true);
                 setLoadingState(false); // Re-enable button on error
             });
     });
 
     // --- Helper Functions ---
     function displayMessage(message, isError) {
         messageArea.textContent = message;
         messageArea.className = isError ? 'message error' : 'message success'; // Apply CSS class
     }
 
     function setLoadingState(isLoading, message = "") {
         if (isLoading) {
             loginButton.disabled = true;
             displayMessage(message, false); // Show loading message as non-error
         } else {
             loginButton.disabled = false;
             // Optionally clear loading message or leave the last status/error
             // if (messageArea.textContent.includes("Logging in...")) {
             //     displayMessage("", false); // Clear only if it was a loading message
             // }
         }
     }
 
 }); // End DOMContentLoaded