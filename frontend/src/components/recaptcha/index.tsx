import Recaptcha from 'react-recaptcha';
import { Helmet } from 'umi';

interface RecaptchaProps {
  className?: string;
  value?: string;
  onChange?: (value: string) => void;
  innerRef?: React.LegacyRef<Recaptcha>;
  cnmirror: boolean;
  sitekey: string;
}

const ReCAPTCHA: React.FC<RecaptchaProps> = (props) => {
  return (
    <>
      <Helmet>
        <script
          defer
          type="application/javascript"
          src={
            props.cnmirror
              ? 'https://www.recaptcha.net/recaptcha/api.js'
              : 'https://www.google.com/recaptcha/api.js'
          }
        ></script>
      </Helmet>
      <Recaptcha
        render="explicit"
        ref={props.innerRef}
        sitekey={props.sitekey}
        className={props.className}
        onloadCallback={() => {}}
        verifyCallback={(res) => {
          props.onChange?.(res);
        }}
        expiredCallback={() => {
          props.onChange?.('');
        }}
      />
    </>
  );
};

export default ReCAPTCHA;
