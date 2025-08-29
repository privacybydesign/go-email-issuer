import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import LanguageDetector from "i18next-browser-languagedetector";

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    detection: {
      order: ["path", "navigator"],
      lookupFromPathIndex: 0,
    },
    resources: {
      en: {
        translation: {
          index_title: "Adding an e-mail address",
          index_header: "Adding e-mail address",
          index_explanation: "Add your e-mail address in your Yivi app.",
          index_multiple_numbers:
            "Do you want to add multiple e-mail addresses? Then follow these steps for each e-mail address you want to add.",
          email_address: "E-mail address",
          index_start: "Start verification",
          index_email_placeholder: "email@example.com",
          index_email_not_valid: "The entered e-mail address is not valid.",
          validate_header: "Check your e-mail address",
          validate_explanation:
            "Please check the e-mail address again for confirmation. Choose 'Back' to correct your e-mail address.",
          back: "Back",
          confirm: "Confirm",
          error_email_format:
            "You did not enter a valid e-mail address. Please check whether the e-mail address is correct.",
          error_internal:
            "Internal error. Please contact Yivi if this happens more often.",
          error_sending_email:
            "Sending the e-mail fails. Most likely this is problem in the Yivi system. Please contact Yivi if this happens more often.",
          error_ratelimit:
            "You have requested to many times. Please try again after {{time}}.",
          error_link_expired:
            "The verification link has expired. Please start the process again.",
          error_token_invalid: "The verification code is invalid.",
          verify: "Verify",
          receive_email: "You will receive an email from Yivi.",
          steps: "Take the following steps:",
          step_1: "Open the email sent by Yivi.",
          step_2: "Enter the verification code in the email or click the link.",
          step_3:
            "Scan the QR code in your Yivi app. On mobile, you can just open the Yivi app to continue.",
          step_4: "Complete the steps on your app to add the email address.",
          verification_code: "Verification code",
          sending_email: "E-mail is being sent...",
          email_sent: "E-mail has been sent.",
          error_header: "Error message",
          error_default:
            "An unknown error has occurred. Please try again later.",
          email_add_success: "E-mail address added.",
          email_add_cancel: "Cancelled.",
          email_add_error:
            "Unfortunately, it was not possible to add this email address to the Yivi app.",
          enter_verification_code: "Enter verification code",
          error_invalid_token: "The verification code is invalid.",
          done_header: "Email address added",
          thank_you: "Thank you for using Yivi, you can close this page now.",
          again: "Add another e-mail address",
        },
      },
      nl: {
        translation: {
          index_title: "E-mailadres toevoegen",
          index_header: "E-mailadres toevoegen",
          index_explanation: "Zet je e-mailadres in je Yivi-app.",
          index_multiple_numbers:
            "Wil je meerdere e-mailadressen toevoegen? Doorloop deze stappen dan voor elk e-mailadres dat je wilt toevoegen.",
          email_address: "E-mailadres",
          index_start: "Start verificatie",
          index_email_placeholder: "email@example.com",
          index_email_not_valid: "Het ingevoerde e-mailadres is niet geldig.",
          validate_header: "E-mailadres controleren",
          validate_explanation:
            "Controleer het e-mailadres nogmaals. Kies 'Terug' om je e-mailadres te corrigeren.",
          back: "Terug",
          confirm: "Bevestigen",
          error_email_address_format:
            "Je hebt geen geldig e-mailadres ingevoerd. Controleer of het ingevoerde e-mailadres klopt.",
          error_internal:
            "Interne fout. Neem contact op met Yivi als dit vaker voorkomt.",
          error_sending_email:
            "De e-mail kan niet worden verzonden. Dit is waarschijnlijk een probleem in Yivi. Neem contact op met Yivi als dit vaker voorkomt.",
          error_ratelimit:
            "U heeft te vaak een verzoek gedaan. Probeer het opnieuw na {{time}}.",
          error_link_expired:
            "De verificatielink is verlopen. Start het proces opnieuw.",
          error_token_invalid: "De verificatiecode is ongeldig.",
          verify: "Verifiëren",
          receive_email: "Je ontvangt een e-mail van Yivi.",
          steps: "Doorloop de volgende stappen:",
          step_1: "Open het email-bericht afkomstig van Yivi.",
          step_2: "Voer de verificatiecode in de email in of klik op de link.",
          step_3:
            "Scan de QR-code in je Yivi-app. Op mobiel kan je de Yivi-app openen om door te gaan.",
          step_4:
            "Voltooi de stappen in je app om het emailadres toe te voegen.",
          verification_code: "Verificatiecode",
          sending_email: "Email wordt verstuurd...",
          email_sent: "Email is verstuurd.",
          enter_verification_code: "Voer verificatiecode in",
          verifying_token: "Code wordt geverifieerd ...",
          error_header: "Foutmelding",
          error_default:
            "Er is een onbekende fout opgetreden. Probeer het later opnieuw.",
          email_add_success: "E-mailadres toegevoegd.",
          email_add_cancel: "Geannuleerd.",
          email_add_error:
            "Het is helaas niet gelukt dit emailadres toe te voegen aan de Yivi-app.",
          error_invalid_token: "De verificatiecode is ongeldig.",
          done_header: "Emailadres toegevoegd",
          thank_you:
            "Bedankt voor het gebruiken van Yivi, u kunt deze pagina nu sluiten.",
          again: "Nog een e-mailadres toevoegen",
        },
      },
    },
    lng: "nl", // default language
    fallbackLng: "en",

    interpolation: {
      escapeValue: false, // react already escapes
    },
  });

export default i18n;
