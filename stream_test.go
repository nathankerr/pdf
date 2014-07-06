package pdf

import (
	"bytes"
	"testing"
)

//§ 7.4.1
func TestFilterExample3(t *testing.T) {
	t.Skip()

	indirectObjectString := "1 0 obj\n<< /Length 534\n/Filter [/ASCII85Decode /LZWDecode] >>\nstream\nJ..)6T`?p&<!J9%_[umg\"B7/Z7KNXbN'S+,*Q/&\"OLT'F\nLIDK#!n`$\"<Atdi`\\Vn%b%)&'cA*VnK\\CJY(sF>c!Jnl@\nRM]WM;jjH6Gnc75idkL5]+cPZKEBPWdR>FF(kj1_R%W_d\n&/jS!;iuad7h?[L-F$+]]0A3Ck*$I0KZ?;<)CJtqi65Xb\nVc3\\n5ua:Q/=0$W<#N3U;H,MQKqfg1?:lUpR;6oN[C2E4\nZNr8Udn.'p+?#X+1>0Kuk$bCDF/(3fL5]Oq)^kJZ!C2H1\n'TO]Rl?Q:&'<5&iP!$Rq;BXRecDN[IJB`,)o8XJOSJ9sD\nS]hQ;Rj@!ND)bD_q&C\\g:inYC%)&u#:u,M6Bm%IY!Kb1+\n\":aAa'S`ViJglLb8<W9k6Yl\\\\0McJQkDeLWdPN?9A'jX*\nal>iG1p&i;eVoK&juJHs9%;Xomop\"5KatWRT\"JQ#qYuL,\nJD?M$0QP)lKn06l1apKDC@\\qJ4B!!(5m+j.7F790m(Vj8\n8l8Q:_CZ(Gm1%X\\N1&u!FKHMB~>\nendstream\nendobj"
	expectedStream := []byte(`2J
BT
/F1 12 Tf 0 Tc
0 Tw
72.5 712 TD
[(Unfiltered streams can be read easily) 65 (, )] TJ
0 -14 TD
[(b) 20 (ut generally tak) 10 (e more space than \311)] TJ
T* (compressed streams.) Tj
0 -28 TD
[(Se) 25 (v) 15 (eral encoding methods are a) 20 (v) 25 (ailable in PDF) 80 (.)] TJ 0 -14 TD
(Some are used for compression and others simply) Tj
T* [(to represent binary data in an ) 55 (ASCII format.)] TJ
T* (Some of the compression filters are \
suitable ) Tj
T* (for both data and images, while others are \
suitable only ) Tj
T* (for continuous-tone images.) Tj
ET`)

	object, _, err := parseIndirectObject([]byte(indirectObjectString))
	if err != nil {
		t.Fatal(err)
	}

	indirectObject := object.(IndirectObject)
	stream := indirectObject.Object.(Stream)

	decodedStream, err := stream.Decode()
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(decodedStream, expectedStream) != 0 {
		t.Errorf("Stream did not decode, got:\n\t%vexpected:\n\t%v", stream, expectedStream)
	}
}
