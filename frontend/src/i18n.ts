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
          index_title: "Add email address",
          index_header: "Add email address",
          index_explanation: "Add your email address in your Yivi app.",
          index_multiple_numbers:
            "Do you want to add multiple email addresses? Then follow these steps for each email address you want to add.",
          email: "Email address",
          index_start: "Start verification",
          index_email_placeholder: "email@example.com",
          index_email_not_valid: "The entered email address is not valid.",
          validate_bot_control:
            "The system is validating that you are not a bot. Please wait.",
          validate_header: "Check your email address",
          validate_explanation:
            "Please check the email address again for confirmation. Choose 'Back' to correct your email address.",
          back: "Back",
          confirm: "Confirm",
          error_email_format:
            "You did not enter a valid email address. Please check whether the email address is correct.",
          error_internal:
            "Internal error. Please contact Yivi if this happens more often.",
          error_sending_email:
            "Sending the email fails. Most likely this is problem in the Yivi system. Please contact Yivi if this happens more often.",
          error_ratelimit:
            "You have requested to many times. Please try again after {{time}}.",
          error_cannot_validate_token:
            "The code cannot be verified. Is there a typo?",
          error_captcha_failed: "Captcha verification failed.",
          error_invalid_captcha:
            "Captcha verification failed. Please try again.",
          error_address_malformed:
            "The email address you entered is not supported by us. You can only add (European) email addresses.",
          verify: "Verify",
          receive_email: "You will receive an email from Yivi.",
          steps: "Take the following steps:",
          step_1: "Open the email sent by Yivi.",
          step_2: "Follow the link in the email.",
          step_3: "You will be redirected back to your Yivi app.",
          not_mobile:
            "Are you not viewing this page on your mobile? Then enter the verification code from the email below.",
          verification_code: "Verification code",
          sending_email: "Email is being sent...",
          email_sent: "Email has been sent.",
          verifying_token: "Code is being verified ...",
          error_header: "Error message",
          error_default:
            "An unknown error has occurred. Please try again later.",
          email_add_success: "Email address added.",
          email_add_cancel: "Cancelled.",
          email_add_error:
            "Unfortunately, it was not possible to add this email address to the Yivi app.",
          done_header: "Email address added",
          thank_you: "Thank you for using Yivi, you can close this page now.",
          again: "Add another email address",
        },
      },
      nl: {
        translation: {
          index_title: "Emailadres toevoegen",
          index_header: "Emailadres toevoegen",
          index_explanation: "Zet je emailadres in je Yivi-app.",
          index_multiple_numbers:
            "Wil je meerdere emailadressen toevoegen? Doorloop deze stappen dan voor elk emailadres dat je wilt toevoegen.",
          email_address: "Emailadres",
          index_start: "Start verificatie",
          index_email_placeholder: "email@example.com",
          index_email_not_valid: "Het ingevoerde emailadres is niet geldig.",
          validate_bot_control:
            "Het systeem controleert of je geen bot bent. Even geduld alstublieft.",
          validate_header: "Emailadres controleren",
          validate_explanation:
            "Controleer het emailadres nogmaals ter bevestiging. Kies 'Terug' om je emailadres te corrigeren.",
          back: "Terug",
          confirm: "Bevestigen",
          error_email_address_format:
            "Je hebt geen geldig emailadres ingevoerd. Controleer of het ingevoerde emailadres klopt.",
          error_internal:
            "Interne fout. Neem contact op met Yivi als dit vaker voorkomt.",
          error_sending_email:
            "De email kan niet worden verzonden. Dit is waarschijnlijk een probleem in Yivi. Neem contact op met Yivi als dit vaker voorkomt.",
          error_ratelimit:
            "U heeft te vaak een verzoek gedaan. Probeer het opnieuw na {{time}}.",
          error_cannot_validate_token:
            "De code kon niet worden geverifieerd. Zit er geen typfout in?",
          error_captcha_failed: "Captcha verificatie mislukt.",
          error_invalid_captcha:
            "Captcha verificatie mislukt. Probeer het opnieuw.",
          error_address_malformed:
            "Het ingevoerde emailadres wordt niet ondersteund. Je kan alleen (Europese) emailadressen toevoegen.",
          verify: "Verifiëren",
          receive_email: "Je ontvangt een email van Yivi.",
          steps: "Doorloop de volgende stappen:",
          step_1: "Open het email-bericht afkomstig van Yivi.",
          step_2: "Kies de link in het email-bericht.",
          step_3: "Je wordt teruggestuurd naar je Yivi-app.",
          not_mobile:
            "Bekijk je deze pagina niet op je mobiel? Vul hieronder de verificatiecode uit het email-bericht in.",
          verification_code: "Verificatiecode",
          sending_email: "Email wordt verstuurd...",
          email_sent: "Email is verstuurd.",
          verifying_token: "Code wordt geverifieerd ...",
          error_header: "Foutmelding",
          error_default:
            "Er is een onbekende fout opgetreden. Probeer het later opnieuw.",
          email_add_success: "Emailadres toegevoegd.",
          email_add_cancel: "Geannuleerd.",
          email_add_error:
            "Het is helaas niet gelukt dit emailadres toe te voegen aan de Yivi-app.",
          done_header: "Emailadres toegevoegd",
          thank_you:
            "Bedankt voor het gebruik van Yivi, u kunt deze pagina nu sluiten.",
          again: "Nog een emailadres toevoegen",
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
