import { Helmet } from '@umijs/max';
import Recaptcha from 'react-recaptcha';

export interface RecaptchaProps {
  value?: string;
  onChange?: (value: string) => void;
  innerRef?: React.LegacyRef<Recaptcha>;
  url: string;
  sitekey: string;
}

const ReCAPTCHA: React.FC<RecaptchaProps> = (props) => {
  return (
    <>
      <Helmet>
        <script
          defer
          type="application/javascript"
          src={props.url + '/recaptcha/api.js'}
        ></script>
      </Helmet>
      <Recaptcha
        render="explicit"
        ref={props.innerRef}
        sitekey={props.sitekey}
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
